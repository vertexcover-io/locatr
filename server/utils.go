package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"

	"gopkg.in/validator.v2"
)

func validateIncomingMessage(message incomingMessage) error {
	inputMessageValidator := validator.NewValidator()

	if err := inputMessageValidator.Validate(message); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if message.Type == "initial_handshake" {
		if (message.Settings == locatrSettings{}) {
			return fmt.Errorf("missing locatrSettings for 'initial_handshake' type")
		}

		if message.Settings.PluginType == "selenium" {
			if message.Settings.SeleniumUrl == "" {
				return fmt.Errorf("selenium plugin type selected but 'selenium_url' is missing or empty")
			}
			if message.Settings.SeleniumSessionId == "" {
				return fmt.Errorf("selenium plugin type selected but 'selenium_session_id' is missing or empty")
			}
		} else if message.Settings.PluginType == "cdp" {
			if message.Settings.CdpURl == "" {
				return fmt.Errorf("cdp plugin type selected but 'cdp_url' is missing or empty")
			}
		} else {
			return fmt.Errorf("invalid 'plugin_type' provided: '%s'. Expected 'selenium' or 'cdp'", message.Settings.PluginType)
		}
	} else if message.Type == "locatr_request" {
		if message.UserRequest == "" {
			return fmt.Errorf("empty 'Input' field provided")
		}
	}
	return nil
}

func dumpJson(inputStruct interface{}) []byte {
	bytesString, _ := json.Marshal(inputStruct)
	return bytesString
}

func generateBytesMessage(outputMessage outgoingMessage) []byte {
	buf := new(bytes.Buffer)
	bytesString := dumpJson(outputMessage)
	length := len(bytesString)
	_ = binary.Write(buf, binary.BigEndian, int32(length))
	buf.Write(bytesString)
	return buf.Bytes()
}

func handleReadError(err error) {
	if err == io.EOF {
		log.Println("Connection closed by client")
	} else {
		log.Printf("Failed to read message: %v", err)
	}
}

func writeResponse(fd net.Conn, msg outgoingMessage) error {
	data := generateBytesMessage(msg)
	_, err := fd.Write(data)
	if err != nil {
		log.Printf("Error writing response: %v", err)
		return err
	}
	return nil
}
