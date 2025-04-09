package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"

	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/playwright-community/playwright-go"
	locatr "github.com/vertexcover-io/locatr/pkg"
	"github.com/vertexcover-io/locatr/pkg/llm"
	"github.com/vertexcover-io/locatr/pkg/logging"
	"github.com/vertexcover-io/locatr/pkg/plugins"
	"github.com/vertexcover-io/locatr/pkg/types"
	"github.com/vertexcover-io/selenium"
)

var VERSION = []uint8{0, 0, 1}

var clientAndLocatrs = make(map[string]*locatr.Locatr)

func handleLocatrRequest(message incomingMessage) (*types.LocatrCompletion, error) {
	locatr, ok := clientAndLocatrs[message.ClientId]
	if !ok {
		return nil, fmt.Errorf("%v of id: %s", ErrClientNotInstantiated, message.ClientId)
	}
	completion, err := locatr.Locate(message.UserRequest)
	if err != nil {
		return nil, fmt.Errorf("%v: %w", ErrFailedToRetrieveLocatr, err)
	}
	return &completion, nil

}

func handleInitialHandshake(message incomingMessage, logger *slog.Logger) error {
	var (
		plugin types.PluginInterface
		err    error
	)
	settings := message.Settings
	switch settings.PluginType {
	case "cdp":
		pw, err := playwright.Run()
		if err != nil {
			return fmt.Errorf("could not start Playwright: %v", err)
		}
		browser, err := pw.Chromium.ConnectOverCDP(settings.CdpURl)
		if err != nil {
			return fmt.Errorf("could not connect to browser: %v", err)
		}
		plugin, err = plugins.NewPlaywrightPlugin(&browser.Contexts()[0].Pages()[0])
		if err != nil {
			return fmt.Errorf("could not create playwright plugin: %v", err)
		}
	case "selenium":
		driver, err := selenium.ConnectRemote(settings.SeleniumUrl, settings.SeleniumSessionId)
		if err != nil {
			return fmt.Errorf("unable to connect to remote selenium instance: %w", err)
		}
		plugin, err = plugins.NewSeleniumPlugin(&driver)
		if err != nil {
			return fmt.Errorf("could not create selenium plugin: %v", err)
		}
	case "appium":
		plugin, err = plugins.NewAppiumPlugin(settings.AppiumUrl, settings.AppiumSessionId)
		if err != nil {
			return fmt.Errorf("unable to create appium plugin: %w", err)
		}
	}
	llmSettings := settings.LlmSettings
	llmClient, err := llm.NewLLMClient(
		llm.WithProvider(types.LLMProvider(llmSettings.LlmProvider)),
		llm.WithModel(llmSettings.ModelName),
		llm.WithAPIKey(llmSettings.LlmApiKey),
	)
	if err != nil {
		return fmt.Errorf("unable to create llm client: %w", err)
	}
	if llmSettings.ReRankerApiKey != "" {
		os.Setenv("COHERE_API_KEY", llmSettings.ReRankerApiKey)
	}

	options := []locatr.Option{
		locatr.WithLLMClient(llmClient), locatr.WithLogger(logger),
	}
	if settings.UseCache {
		var path *string = nil // This will tell locatr to use the default cache path
		if settings.CachePath != "" {
			path = &settings.CachePath
		}
		options = append(options, locatr.EnableCache(path))
	}
	locatr, err := locatr.NewLocatr(plugin, options...)
	if err != nil {
		return fmt.Errorf("unable to create locatr instance: %w", err)
	}

	clientAndLocatrs[message.ClientId] = locatr
	return nil
}

func acceptConnection(fd net.Conn, logger *slog.Logger) {
	lengthBuf := make([]byte, 4)
	versionBuf := make([]byte, 3)
	for {
		sum := 0
		count, err := fd.Read(versionBuf)
		if err != nil {
			handleReadError(err, logger)
			return
		}
		sum += count
		if !(compareVersion(versionBuf)) {
			msg := outgoingMessage{
				Status: "error",
				Error: fmt.Sprintf(
					"%v client version: %s, server version: %s",
					ErrClientAndServerVersionMisMatch,
					getVersionString(convertVersionToUint8(versionBuf)),
					getVersionString(VERSION),
				),
			}
			if err := writeResponse(fd, msg, logger); err != nil {
				logger.Error("Failed to send error response to client",
					"error", err)
			}
			return
		}

		count, err = fd.Read(lengthBuf)
		if err != nil {
			handleReadError(err, logger)
			return
		}
		msgLength := binary.BigEndian.Uint32(lengthBuf)
		sum += count

		message := make([]byte, msgLength)
		count, err = fd.Read(message)
		if err != nil {
			handleReadError(err, logger)
			return
		}
		sum += count

		logger.Debug("Read bytes from client", slog.Int("count", sum))

		var clientMessage incomingMessage
		if err := json.Unmarshal(message, &clientMessage); err != nil {
			logger.Error("Error parsing JSON",
				"error", err,
				"message", string(message))
			msg := outgoingMessage{
				Type:     "error",
				Status:   "error",
				ClientId: "00000000-0000-0000-0000-000000000000",
				Error:    err.Error(),
			}
			if err := writeResponse(fd, msg, logger); err != nil {
				logger.Error("Failed to send error response to client",
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
			if err := writeResponse(fd, errResp, logger); err != nil {
				logger.Error("Failed to send validation error response to client",
					"error", err)
				return
			}
			continue
		}

		defer delete(clientAndLocatrs, clientMessage.ClientId)

		switch clientMessage.Type {
		case "initial_handshake":
			err := handleInitialHandshake(clientMessage, logger)
			if err != nil {
				errResp := outgoingMessage{
					Type:     clientMessage.Type,
					Status:   "error",
					Error:    err.Error(),
					ClientId: clientMessage.ClientId,
				}
				if err := writeResponse(fd, errResp, logger); err != nil {
					logger.Error("Failed to send error response to client during handshake",
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
			if err := writeResponse(fd, successResp, logger); err != nil {
				logger.Error("Failed to send success response to client during handshake",
					"error", err,
					"clientId", clientMessage.ClientId)
				return
			}
			logger.Info("Initial handshake successful with client",
				"clientId", clientMessage.ClientId)
		case "locatr_request":
			locatrOutput, err := handleLocatrRequest(clientMessage)
			if err != nil {
				errResp := outgoingMessage{
					Type:      clientMessage.Type,
					Selectors: []string{},
					Status:    "error",
					Error:     err.Error(),
					ClientId:  clientMessage.ClientId,
				}
				if err := writeResponse(fd, errResp, logger); err != nil {
					logger.Error("Failed to send error response to client during locatr request",
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
				Selectors:    locatrOutput.Locators,
				SelectorType: string(locatrOutput.LocatorType),
			}
			if err := writeResponse(fd, successResp, logger); err != nil {
				logger.Error("Failed to send success response to client during locatr request",
					"error", err,
					"clientId", clientMessage.ClientId)
				return
			}
		}
	}
}

func main() {
	var socketFilePath string
	var logLevel int
	flag.StringVar(&socketFilePath, "socketFilePath", "/tmp/locatr.sock", "path to the socket file to listen at.")
	flag.IntVar(&logLevel, "logLevel", int(slog.LevelError), "log level for the server")
	flag.Parse()

	if _, err := os.Stat(socketFilePath); !errors.Is(err, os.ErrNotExist) {
		os.Remove(socketFilePath)
	}
	var logger *slog.Logger
	if (logLevel == int(slog.LevelDebug)) ||
		(logLevel == int(slog.LevelInfo)) ||
		(logLevel == int(slog.LevelWarn)) ||
		(logLevel == int(slog.LevelError)) {
		logFile, err := os.Create("locatr-logs.jsonl")
		if err != nil {
			log.Fatalf("unable to create log file: %v", err)
		}
		logger = logging.NewLogger(slog.Level(logLevel), logFile)
	}

	socket, err := net.Listen("unix", socketFilePath)
	if err != nil {
		logger.Error("Failed connecting to socket", "error", err)
		os.Exit(1)
	}
	defer socket.Close()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Info("Received signal, shutting down...", "signal", sig)

		if err := os.Remove(socketFilePath); err != nil {
			logger.Error("Failed to remove socket file", "error", err)
		}
		os.Exit(0)
	}()

	logging.DefaultLogger.Info("Ready to accept connections", "socketFilePath", socketFilePath)
	defer os.Remove(socketFilePath)

	for {
		client, err := socket.Accept()
		if err != nil {
			logger.Error("Failed accepting socket", "error", err)
			continue
		}
		go func() {
			acceptConnection(client, logger)
			client.Close()
		}()
	}
}
