package tracing

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel"
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

	opts = append(opts, otlptracegrpc.WithEndpoint(endpoint))
	if insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptracegrpc.New(
		ctx,
		opts...,
	)
	if err != nil {
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
		return nil, fmt.Errorf("failed to create resource: %w: %w", ErrOtelTraceProvider, err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(r),
	)
	return tp, nil
}

func SetupOtelSDK(ctx context.Context, opts ...Option) (OtelShutdownFunc, error) {
	var shutdown OtelShutdownFunc

	cfg := getConfig(opts)

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

func GetTraceProvider() trace.TracerProvider {
	tp := otel.GetTracerProvider()
	if tp == nil {
		panic("ensure setup is called before fetching trace provider")
	}
	return tp
}
