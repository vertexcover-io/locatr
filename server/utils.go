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
		} else {
			return fmt.Errorf("%v: '%s'. Expected 'selenium' or 'cdp'", ErrInvalidPluginType, message.Settings.PluginType)
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

func generateBytesMessage(outputMessage outgoingMessage) []byte {
	buf := new(bytes.Buffer)
	bytesString := dumpJson(outputMessage)
	length := len(bytesString)
	for _, val := range VERSION {
		err := binary.Write(buf, binary.BigEndian, val)
		if err != nil {
			log.Fatalf("Error writing version to buffer %v", err)
			break
		}
	}
	err := binary.Write(buf, binary.BigEndian, int32(length))
	if err != nil {
		log.Fatalf("Error writing length to buffer: %v", err)
	}
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
