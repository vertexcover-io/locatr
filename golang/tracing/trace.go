package tracing

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/vertexcover-io/locatr/golang/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

var ErrOtelGRPCExporter = errors.New("OTEL GRPC exporter failed")
var ErrOtelTraceProvider = errors.New("OTEL Trace Provider create failed")

type OtelShutdownFunc func(context.Context) error

func NewGRPCExporter(ctx context.Context, endpoint string, insecure bool) (*otlptrace.Exporter, error) {
	var opts []otlptracegrpc.Option

	logger.Logger.Debug("setting up GRPC exporter")

	opts = append(opts, otlptracegrpc.WithEndpoint(endpoint))
	if insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptracegrpc.New(
		ctx,
		opts...,
	)
	if err != nil {
		logger.Logger.Warn("failed to create OTLP gRPC exporter")
		return nil, fmt.Errorf("failed to create: %w: %w", ErrOtelGRPCExporter, err)
	}

	return exporter, nil
}

func NewTraceProvider(exp sdktrace.SpanExporter, svcName string) (*sdktrace.TracerProvider, error) {
	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(svcName),
		),
	)
	if err != nil {
		logger.Logger.Warn("failed to create Otel Trace Provider")
		return nil, fmt.Errorf("failed to create resource: %w: %w", ErrOtelTraceProvider, err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(r),
	)
	return tp, nil
}

var setupRun bool = false

func SetupOtelSDK(ctx context.Context, opts ...Option) (OtelShutdownFunc, error) {
	var shutdown OtelShutdownFunc

	cfg := getConfig(opts)

	logger.Logger.Debug(
		"setting up Open telemetry SDK",
		slog.String("endpoint", cfg.endpoint),
		slog.String("svc-name", cfg.svcName),
		slog.Bool("insecure", cfg.insecure),
	)

	exp, err := NewGRPCExporter(ctx, cfg.endpoint, cfg.insecure)
	if err != nil {
		return shutdown, err
	}

	tp, err := NewTraceProvider(exp, cfg.svcName)
	if err != nil {
		return shutdown, err
	}
	otel.SetTracerProvider(tp)
	shutdown = tp.Shutdown

	logger.Logger.Info(
		"Open Telemetry SDK setup complete",
	)

	setupRun = true

	return shutdown, err
}

func getConfig(opts []Option) config {
	if len(opts) == 0 {
		opts = append(opts, WithDefaults())
	}

	var cfg config
	for _, opt := range opts {
		cfg = *opt.applyOption(&cfg)
	}
	return cfg
}

func GetTracer() trace.Tracer {
	tp := otel.GetTracerProvider()
	if tp == nil {
		panic("ensure setup is called before fetching trace provider")
	}
	return tp.Tracer(DEFAULT_TRACE_NAME)
}

func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if setupRun {
		tracer := GetTracer()
		return tracer.Start(ctx, name, opts...)
	}
	return ctx, &noopSpan{}
}

type noopSpan struct {
	trace.Span
}

func (n *noopSpan) End(opt ...trace.SpanEndOption) {}

func (n *noopSpan) AddEvent(name string, opt ...trace.EventOption) {}

func (n *noopSpan) AddLink(link trace.Link) {}

func (n *noopSpan) IsRecording() bool { return false }

func (n *noopSpan) RecordError(err error, opt ...trace.EventOption) {}

func (n *noopSpan) SpanContext() trace.SpanContext {
	return trace.NewSpanContext(trace.SpanContextConfig{})
}

func (n *noopSpan) SetStatus(code codes.Code, description string) {}

func (n *noopSpan) SetName(name string) {}

func (n *noopSpan) SetAttributes(kv ...attribute.KeyValue) {}

func (n *noopSpan) TracerProvider() trace.TracerProvider { return &noopTracerProvider{} }

type noopTracerProvider struct {
	trace.TracerProvider
}
