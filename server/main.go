package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"

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
	if err != nil {
		return locatr.BaseLocatrOptions{}, fmt.Errorf("%v : %v", FailedToCreateLlmClient, err)
	}
	opts.LlmClient = llmClient

	return opts, nil
}

func handleLocatrRequest(message incomingMessage) (*locatr.LocatrOutput, error) {
	baseLocatr, ok := clientAndLocatrs[message.ClientId]
	if !ok {
		return nil, fmt.Errorf("%v of id: %s", ErrClientNotInstantiated, message.ClientId)
	}
	locatr, err := baseLocatr.GetLocatrStr(message.UserRequest)
	if err != nil {
		return nil, fmt.Errorf("%v: %w", ErrFailedToRetrieveLocatr, err)
	}
	return locatr, nil

}

func handleInitialHandshake(message incomingMessage) error {
	baseLocatrOpts, err := createLocatrOptions(message)
	if err != nil {
		return err
	}
	switch message.Settings.PluginType {
	case "cdp":
		parsedUrl, _ := url.Parse(message.Settings.CdpURl)
		port, _ := strconv.Atoi(parsedUrl.Port())
		connectionOpts := cdpLocatr.CdpConnectionOptions{
			Port:     port,
			HostName: parsedUrl.Hostname(),
		}
		connection, err := cdpLocatr.CreateCdpConnection(connectionOpts)
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
		seleniumLocatr, err := seleniumLocatr.NewRemoteConnSeleniumLocatr(settings.SeleniumUrl, settings.SeleniumSessionId, baseLocatrOpts)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrSeleniumLocatrCreation, err)
		}
		clientAndLocatrs[message.ClientId] = seleniumLocatr
	case "appium":
		settings := message.Settings
		appiumLocatr, err := appiumLocatr.NewAppiumLocatr(settings.AppiumUrl, settings.AppiumSessionId, baseLocatrOpts)
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
		_, err := fd.Read(versionBuf)
		if err != nil {
			handleReadError(err)
			return
		}
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
		_, err = fd.Read(lengthBuf)
		msgLength := binary.BigEndian.Uint32(lengthBuf)
		if err != nil {
			handleReadError(err)
			return
		}
		message := make([]byte, msgLength)
		_, err = fd.Read(message)
		if err != nil {
			handleReadError(err)
			return
		}

		var clientMessage incomingMessage
		if err := json.Unmarshal(message, &clientMessage); err != nil {
			logger.Logger.Error("Error parsing JSON",
				"error", err,
				"message", string(message))
			msg := outgoingMessage{
				Type:     clientMessage.Type,
				Status:   "error",
				ClientId: clientMessage.ClientId,
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

		switch clientMessage.Type {
		case "initial_handshake":
			err := handleInitialHandshake(clientMessage)
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
			locatrOutput, err := handleLocatrRequest(clientMessage)
			if err != nil {
				errResp := outgoingMessage{
					Type:     clientMessage.Type,
					Status:   "error",
					Error:    err.Error(),
					ClientId: clientMessage.ClientId,
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

func main() {
	var socketFilePath string
	var logLevel int
	flag.StringVar(&socketFilePath, "socketFilePath", "/tmp/locatr.sock", "path to the socket file to listen at.")
	flag.IntVar(&logLevel, "logLevel", int(slog.LevelError), "log level for the server")
	flag.Parse()

	if _, err := os.Stat(socketFilePath); !errors.Is(err, os.ErrNotExist) {
		os.Remove(socketFilePath)
	}
	if (logLevel == int(slog.LevelDebug)) ||
		(logLevel == int(slog.LevelInfo)) ||
		(logLevel == int(slog.LevelWarn)) ||
		(logLevel == int(slog.LevelError)) {
		logger.Level.Set(slog.Level(logLevel))
	}

	socket, err := net.Listen("unix", socketFilePath)
	if err != nil {
		logger.Logger.Error("Failed connecting to socket", "error", err)
		os.Exit(1)
	}
	defer socket.Close()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Logger.Info("Received signal, shutting down...", "signal", sig)

		if err := os.Remove(socketFilePath); err != nil {
			logger.Logger.Error("Failed to remove socket file", "error", err)
		}
		os.Exit(0)
	}()

	logger.Logger.Info("Ready to accept connections", "socketFilePath", socketFilePath)
	defer os.Remove(socketFilePath)

	for {
		client, err := socket.Accept()
		if err != nil {
			logger.Logger.Error("Failed accepting socket", "error", err)
			continue
		}
		go func() {
			acceptConnection(client)
			client.Close()
		}()
	}
}
