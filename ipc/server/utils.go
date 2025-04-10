package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"

	"gopkg.in/validator.v2"
)

func validateIncomingMessage(message incomingMessage) error {
	inputMessageValidator := validator.NewValidator()

	if err := inputMessageValidator.Validate(message); err != nil {
		return fmt.Errorf("%v: %w", ErrInputMessageValidationFailed, err)
	}

	if message.Type == "initial_handshake" {
		if (message.Settings == locatrSettings{}) {
			return ErrMissingLocatrSettings
		}

		if message.Settings.PluginType == "selenium" {
			if message.Settings.SeleniumUrl == "" {
				return ErrMissingSeleniumUrl
			}
			if message.Settings.SeleniumSessionId == "" {
				return ErrMissingSeleniumSessionId
			}
		} else if message.Settings.PluginType == "cdp" {
			if message.Settings.CdpURl == "" {
				return ErrMissingCdpUrl
			}
		} else if message.Settings.PluginType == "appium" {
			if message.Settings.AppiumUrl == "" {
				return ErrMissingAppiumUrl
			}
			if message.Settings.AppiumSessionId == "" {
				return ErrMissingAppiumSessionId
			}
		} else {
			return fmt.Errorf("%w: '%s'. Expected 'selenium' or 'cdp' or 'appium'", ErrInvalidPluginType, message.Settings.PluginType)
		}
	} else if message.Type == "locatr_request" {
		if message.UserRequest == "" {
			return ErrEmptyUserRequest
		}
	}
	return nil
}

func dumpJson(inputStruct interface{}) []byte {
	bytesString, _ := json.Marshal(inputStruct)
	return bytesString
}

func generateBytesMessage(outputMessage outgoingMessage, logger *slog.Logger) []byte {
	buf := new(bytes.Buffer)
	bytesString := dumpJson(outputMessage)
	length := len(bytesString)

	for _, val := range VERSION {
		err := binary.Write(buf, binary.BigEndian, val)
		if err != nil {
			logger.Error("Error writing version to buffer", "error", err)
			break
		}
	}

	err := binary.Write(buf, binary.BigEndian, int32(length))
	if err != nil {
		logger.Error("Error writing length to buffer",
			"error", err)
	}
	buf.Write(bytesString)
	return buf.Bytes()
}

func handleReadError(err error, logger *slog.Logger) {
	if err == io.EOF {
		logger.Info("Connection closed by client")
	} else {
		logger.Error("Failed to read message", "error", err)
	}
}

func writeResponse(fd net.Conn, msg outgoingMessage, logger *slog.Logger) error {
	data := generateBytesMessage(msg, logger)
	_, err := fd.Write(data)
	if err != nil {
		logger.Error("Error writing response",
			"error", err,
			"clientId", msg.ClientId) // Assuming msg has ClientId
		return err
	}
	logger.Info("Response written to client",
		"clientId", msg.ClientId, // Assuming msg has ClientId
		"status", msg.Status,
		"type", msg.Type)
	return nil
}

func compareVersion(version []byte) bool {
	for i, ver := range version {
		if uint8(ver) != VERSION[i] {
			return false
		}
	}
	return true
}

func convertVersionToUint8(versionBytes []byte) []uint8 {
	versionIntArray := make([]uint8, 3)
	for i, ver := range versionBytes {
		versionIntArray[i] = uint8(ver)
	}
	return versionIntArray
}
func getVersionString(versionInt []uint8) string {
	versionString := ""
	for i, ver := range versionInt {
		if i > 0 {
			versionString = fmt.Sprintf("%s.%d", versionString, ver)
		} else {
			versionString = fmt.Sprintf("%d", uint8(ver))
		}
	}
	return versionString
}
