package main

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"strconv"

	"github.com/vertexcover-io/locatr"
)

var clientAndLocatrs = make(map[string]locatr.LocatrInterface)

func createLocatrOptions(message incomingMessage) locatr.BaseLocatrOptions {
	opts := locatr.BaseLocatrOptions{}
	llmConfig := message.Settings.LlmSettings

	if llmConfig.ReRankerApiKey != "" {
		opts.ReRankClient = locatr.NewCohereClient(llmConfig.ReRankerApiKey)
	}

	opts.LogConfig = locatr.LogConfig{Level: locatr.Debug}

	opts.CachePath = message.Settings.CachePath
	opts.UseCache = message.Settings.UseCache

	opts.ResultsFilePath = message.Settings.ResultsFilePath

	llmClient, _ := locatr.NewLlmClient(
		locatr.LlmProvider(llmConfig.LlmProvider),
		llmConfig.ModelName,
		llmConfig.LlmApiKey,
	)
	opts.LlmClient = llmClient

	return opts
}

func handleLocatrRequest(message incomingMessage) (string, error) {
	baseLocatr, ok := clientAndLocatrs[message.ClientId]
	if !ok {
		return "", fmt.Errorf("Client not instianciated of id: %s", message.ClientId)
	}
	locatr, err := baseLocatr.GetLocatrStr(message.UserRequest)
	if err != nil {
		return "", fmt.Errorf("Failed to retrieve locatr: %w", err)
	}
	return locatr, nil

}

func handleInitialHandshake(message incomingMessage) error {
	baseLocatrOpts := createLocatrOptions(message)
	switch message.Settings.PluginType {
	case "cdp":
		parsedUrl, _ := url.Parse(message.Settings.CdpURl)
		port, _ := strconv.Atoi(parsedUrl.Port())
		connectionOpts := locatr.CdpConnectionOptions{
			Port:     port,
			HostName: parsedUrl.Hostname(),
		}
		connection, err := locatr.CreateCdpConnection(connectionOpts)
		if err != nil {
			return fmt.Errorf("Error while creating cdp connection: %w", err)
		}
		cdpLocatr, err := locatr.NewCdpLocatr(connection, baseLocatrOpts)
		if err != nil {
			return fmt.Errorf("Error while creating cdp locatr: %w", err)
		}
		clientAndLocatrs[message.ClientId] = cdpLocatr
	case "selenium":
		settings := message.Settings
		seleniumLocatr, err := locatr.NewRemoteConnSeleniumLocatr(settings.SeleniumUrl, settings.SeleniumSessionId, baseLocatrOpts)
		if err != nil {
			return fmt.Errorf("Error while creating selenium locatr: %w", err)
		}
		clientAndLocatrs[message.ClientId] = seleniumLocatr
	}
	return nil
}

func acceptConnection(fd net.Conn) {
	defer fd.Close()
	lengthBuff := make([]byte, 4)
	for {
		_, err := fd.Read(lengthBuff)
		msgLength := binary.BigEndian.Uint32(lengthBuff)
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
			log.Printf("Error parsing JSON: %v", err)
			msg := outgoingMessage{
				Type:     clientMessage.Type,
				Status:   "error",
				ClientId: clientMessage.ClientId,
				Error:    err.Error(),
			}
			if err := writeResponse(fd, msg); err != nil {
				log.Printf("Failed to send error response to client: %v", err)
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
				log.Printf("Failed to send validation error response to client: %v", err)
				return
			}
			continue
		}
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
					log.Printf("Failed to send error response to client during handshake: %v", err)
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
				log.Printf("Failed to send success response to client during handshake: %v", err)
				return
			}
		case "locatr_request":
			locatrString, err := handleLocatrRequest(clientMessage)
			if err != nil {
				errResp := outgoingMessage{
					Type:     clientMessage.Type,
					Status:   "error",
					Error:    err.Error(),
					ClientId: clientMessage.ClientId,
				}
				if err := writeResponse(fd, errResp); err != nil {
					log.Printf("Failed to send error response to client during locatr request: %v", err)
					return
				}
				continue
			}
			successResp := outgoingMessage{
				Type:     clientMessage.Type,
				Status:   "ok",
				ClientId: clientMessage.ClientId,
				Output:   locatrString,
			}
			if err := writeResponse(fd, successResp); err != nil {
				log.Printf("Failed to send success response to client during locatr request: %v", err)
				return
			}
		}
	}
}

func main() {
	var socketFilePath string
	flag.StringVar(&socketFilePath, "socketFilePath", "/tmp/locatr.sock", "path to the socketfile to listen at.")
	flag.Parse()
	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	socket, err := net.Listen("unix", socketFilePath)
	if err != nil {
		log.Fatalf("failed connecting to socket: %v", err)
		return
	}
	log.Printf("Ready to accept connections on file: %s", socketFilePath)
	for {
		fd, err := socket.Accept()
		if err != nil {
			log.Fatal("Failed accepting socket %w", err)
		}
		go acceptConnection(fd)
	}
}
