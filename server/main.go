package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"sync"

	"net"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	locatr "github.com/vertexcover-io/locatr/golang"
	appiumLocatr "github.com/vertexcover-io/locatr/golang/appium"
	cdpLocatr "github.com/vertexcover-io/locatr/golang/cdp"
	"github.com/vertexcover-io/locatr/golang/llm"
	"github.com/vertexcover-io/locatr/golang/logger"
	"github.com/vertexcover-io/locatr/golang/reranker"
	"github.com/vertexcover-io/locatr/golang/seleniumLocatr"
	"github.com/vertexcover-io/locatr/golang/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
)

var VERSION = []uint8{0, 0, 1}

var clientAndLocatrs = make(map[string]locatr.LocatrInterface)

func createLocatrOptions(message incomingMessage) (locatr.BaseLocatrOptions, error) {
	opts := locatr.BaseLocatrOptions{}
	llmConfig := message.Settings.LlmSettings

	if llmConfig.ReRankerApiKey != "" {
		opts.ReRankClient = reranker.NewCohereClient(llmConfig.ReRankerApiKey)
	} else {
		opts.ReRankClient = reranker.CreateCohereClientFromEnv()
	}

	opts.CachePath = message.Settings.CachePath
	opts.UseCache = message.Settings.UseCache

	opts.ResultsFilePath = message.Settings.ResultsFilePath

	llmClient, err := llm.NewLlmClient(
		llm.LlmProvider(llmConfig.LlmProvider),
		llmConfig.ModelName,
		llmConfig.LlmApiKey,
	)
	slog.Debug("Using llm model", slog.String("model", llmConfig.ModelName))

	if err != nil {
		return locatr.BaseLocatrOptions{}, fmt.Errorf("%v : %v", FailedToCreateLlmClient, err)
	}
	opts.LlmClient = llmClient

	return opts, nil
}

func handleLocatrRequest(ctx context.Context, message incomingMessage) (*locatr.LocatrOutput, error) {
	tracer := tracing.GetTracer()

	ctx, span := tracer.Start(ctx, "locator-request")
	defer span.End()

	baseLocatr, ok := clientAndLocatrs[message.ClientId]
	if !ok {
		return nil, fmt.Errorf("%v of id: %s", ErrClientNotInstantiated, message.ClientId)
	}
	locatr, err := baseLocatr.GetLocatrStr(ctx, message.UserRequest)
	if err != nil {
		return nil, fmt.Errorf("%v: %w", ErrFailedToRetrieveLocatr, err)
	}
	return locatr, nil

}

func handleInitialHandshake(ctx context.Context, message incomingMessage) error {
	tracer := tracing.GetTracer()

	ctx, span := tracer.Start(ctx, "initial-handshake")
	defer span.End()

	baseLocatrOpts, err := createLocatrOptions(message)
	if err != nil {
		return err
	}

	span.SetAttributes(
		attribute.String("plugin-type", message.Settings.PluginType),
	)

	switch message.Settings.PluginType {
	case "cdp":
		span.SetAttributes(
			attribute.String("cdp-url", message.Settings.CdpURl),
		)

		parsedUrl, _ := url.Parse(message.Settings.CdpURl)
		port, _ := strconv.Atoi(parsedUrl.Port())
		connectionOpts := cdpLocatr.CdpConnectionOptions{
			Port:     port,
			HostName: parsedUrl.Hostname(),
		}
		connection, err := cdpLocatr.CreateCdpConnection(ctx, connectionOpts)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrCdpConnectionCreation, err)
		}
		cdpLocatr, err := cdpLocatr.NewCdpLocatr(connection, baseLocatrOpts)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrCdpLocatrCreation, err)
		}
		clientAndLocatrs[message.ClientId] = cdpLocatr

	case "selenium":
		settings := message.Settings

		span.SetAttributes(
			attribute.String("selenium-url", settings.SeleniumUrl),
			attribute.String("selenium-session-id", settings.SeleniumSessionId),
		)

		seleniumLocatr, err := seleniumLocatr.NewRemoteConnSeleniumLocatr(
			ctx,
			settings.SeleniumUrl,
			settings.SeleniumSessionId,
			baseLocatrOpts,
		)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrSeleniumLocatrCreation, err)
		}
		clientAndLocatrs[message.ClientId] = seleniumLocatr

	case "appium":
		settings := message.Settings

		span.SetAttributes(
			attribute.String("appium-url", settings.AppiumUrl),
			attribute.String("appium-session-id", settings.AppiumSessionId),
		)

		appiumLocatr, err := appiumLocatr.NewAppiumLocatr(
			ctx,
			settings.AppiumUrl,
			settings.AppiumSessionId,
			baseLocatrOpts,
		)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrAppiumLocatrCreation, err)
		}
		clientAndLocatrs[message.ClientId] = appiumLocatr
	}
	return nil
}

func acceptConnection(fd net.Conn) {
	lengthBuf := make([]byte, 4)
	versionBuf := make([]byte, 3)
	for {
		sum := 0
		count, err := fd.Read(versionBuf)
		if err != nil {
			handleReadError(err)
			return
		}
		sum += count
		if !(compareVersion(versionBuf)) {
			msg := outgoingMessage{
				Status: "error",
				Error: fmt.Sprintf(
					"%v client version: %s, server version: %s",
					ClientAndServerVersionMisMatch,
					getVersionString(convertVersionToUint8(versionBuf)),
					getVersionString(VERSION),
				),
			}
			if err := writeResponse(fd, msg); err != nil {
				logger.Logger.Error("Failed to send error response to client",
					"error", err)
			}
			return
		}

		count, err = fd.Read(lengthBuf)
		if err != nil {
			handleReadError(err)
			return
		}
		msgLength := binary.BigEndian.Uint32(lengthBuf)
		sum += count

		message := make([]byte, msgLength)
		count, err = fd.Read(message)
		if err != nil {
			handleReadError(err)
			return
		}
		sum += count

		logger.Logger.Debug("Read bytes from client", slog.Int("count", sum))

		var clientMessage incomingMessage
		if err := json.Unmarshal(message, &clientMessage); err != nil {
			logger.Logger.Error("Error parsing JSON",
				"error", err,
				"message", string(message))
			msg := outgoingMessage{
				Type:     "error",
				Status:   "error",
				ClientId: "00000000-0000-0000-0000-000000000000",
				Error:    err.Error(),
			}
			if err := writeResponse(fd, msg); err != nil {
				logger.Logger.Error("Failed to send error response to client",
					"error", err)
				return
			}
			continue
		}
		err = validateIncomingMessage(clientMessage)
		if err != nil {
			errResp := outgoingMessage{
				Type:     clientMessage.Type,
				Status:   "error",
				ClientId: clientMessage.ClientId,
				Error:    err.Error(),
			}
			if err := writeResponse(fd, errResp); err != nil {
				logger.Logger.Error("Failed to send validation error response to client",
					"error", err)
				return
			}
			continue
		}

		defer delete(clientAndLocatrs, clientMessage.ClientId)

		var ctx context.Context
		if clientMessage.OtelParentTraceId != "" {
			carrier := propagation.MapCarrier{"traceparent": clientMessage.OtelParentTraceId}

			propagator := propagation.TraceContext{}
			ctx = propagator.Extract(context.Background(), carrier)
		} else {
			ctx = context.Background()
		}

		ctx, span := tracing.StartSpan(ctx, "locator-reqest")
		defer span.End()

		switch clientMessage.Type {
		case "initial_handshake":
			err := handleInitialHandshake(ctx, clientMessage)
			if err != nil {
				errResp := outgoingMessage{
					Type:     clientMessage.Type,
					Status:   "error",
					Error:    err.Error(),
					ClientId: clientMessage.ClientId,
				}
				if err := writeResponse(fd, errResp); err != nil {
					logger.Logger.Error("Failed to send error response to client during handshake",
						"error", err,
						"clientId", clientMessage.ClientId)
					return
				}
				continue
			}
			successResp := outgoingMessage{
				Type:     clientMessage.Type,
				Status:   "ok",
				ClientId: clientMessage.ClientId,
			}
			if err := writeResponse(fd, successResp); err != nil {
				logger.Logger.Error("Failed to send success response to client during handshake",
					"error", err,
					"clientId", clientMessage.ClientId)
				return
			}
			logger.Logger.Info("Initial handshake successful with client",
				"clientId", clientMessage.ClientId)

		case "locatr_request":
			locatrOutput, err := handleLocatrRequest(ctx, clientMessage)
			if err != nil {
				errResp := outgoingMessage{
					Type:      clientMessage.Type,
					Selectors: []string{},
					Status:    "error",
					Error:     err.Error(),
					ClientId:  clientMessage.ClientId,
				}
				if err := writeResponse(fd, errResp); err != nil {
					logger.Logger.Error("Failed to send error response to client during locatr request",
						"error", err,
						"clientId", clientMessage.ClientId)
					return
				}
				continue
			}
			successResp := outgoingMessage{
				Type:         clientMessage.Type,
				Status:       "ok",
				ClientId:     clientMessage.ClientId,
				Selectors:    locatrOutput.Selectors,
				SelectorType: string(locatrOutput.SelectorType),
			}
			if err := writeResponse(fd, successResp); err != nil {
				logger.Logger.Error("Failed to send success response to client during locatr request",
					"error", err,
					"clientId", clientMessage.ClientId)
				return
			}
		}
	}
}

type Config struct {
	SocketFilePath string
	LogLevel       int

	Tracing struct {
		Endpoint    string
		ServiceName string
		Insecure    bool
	}
}

func main() {
	var cfg Config
	flag.StringVar(&cfg.SocketFilePath, "socketFilePath", "/tmp/locatr.sock", "path to the socket file to listen at.")
	flag.IntVar(&cfg.LogLevel, "logLevel", int(slog.LevelError), "log level for the server")
	flag.StringVar(&cfg.Tracing.Endpoint, "tracing.endpoint", tracing.DEFAULT_ENDPOINT, "gRPC endpoint for otel receiver")
	flag.StringVar(&cfg.Tracing.ServiceName, "tracing.svcName", tracing.DEFAULT_SVC_NAME, "name for service to use in Open Telemetry Logs")
	flag.BoolVar(&cfg.Tracing.Insecure, "tracing.insecure", tracing.DEFAULT_INSECURE, "is gRPC endpoint insecure")

	flag.Parse()

	shutdown, err := tracing.SetupOtelSDK(
		context.Background(),
		tracing.WithEndpoint(cfg.Tracing.Endpoint),
		tracing.WithSVCName(cfg.Tracing.ServiceName),
		tracing.WithInsecure(cfg.Tracing.Insecure),
	)
	if err != nil {
		logger.Logger.Error(
			"Failed to setup Open Telemetry SDK",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
	defer func() {
		if sErr := shutdown(context.Background()); sErr != nil {
			err = errors.Join(err, sErr)
			logger.Logger.Error(
				"Error while shutting down Open Telemetry service",
				slog.String("error", err.Error()),
			)
		}
	}()

	if _, err := os.Stat(cfg.SocketFilePath); !errors.Is(err, os.ErrNotExist) {
		os.Remove(cfg.SocketFilePath)
	}
	if (cfg.LogLevel == int(slog.LevelDebug)) ||
		(cfg.LogLevel == int(slog.LevelInfo)) ||
		(cfg.LogLevel == int(slog.LevelWarn)) ||
		(cfg.LogLevel == int(slog.LevelError)) {
		logger.Level.Set(slog.Level(cfg.LogLevel))
	}

	socket, err := net.Listen("unix", cfg.SocketFilePath)
	if err != nil {
		logger.Logger.Error("Failed connecting to socket", "error", err)
		os.Exit(1)
	}
	defer socket.Close()

	ctx := context.Background()

	// there is no manual terminate action
	ctx, _ = signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup

	go func() {
		sig := <-ctx.Done()

		logger.Logger.Info("Received signal, shutting down...", "signal", sig)
		wg.Wait()

		if err := os.Remove(cfg.SocketFilePath); err != nil {
			logger.Logger.Error("Failed to remove socket file", "error", err)
		}
	}()

	logger.Logger.Info("Ready to accept connections", "socketFilePath", cfg.SocketFilePath)
	defer os.Remove(cfg.SocketFilePath)

	for {
		client, err := socket.Accept()
		if err != nil {
			logger.Logger.Error("Failed accepting socket", "error", err)
			continue
		}
		wg.Add(1)
		go func() {
			defer client.Close()
			defer wg.Done()

			acceptConnection(client)
		}()
	}
}
