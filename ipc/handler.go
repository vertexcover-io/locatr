package ipc

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
)

var (
	debugLogger   = log.New(os.Stdout, "DEBUG: ", log.LstdFlags)
	infoLogger    = log.New(os.Stdout, "INFO: ", log.LstdFlags)
	warningLogger = log.New(os.Stdout, "WARNING: ", log.LstdFlags)
	errorLogger   = log.New(os.Stdout, "ERROR: ", log.LstdFlags)
)

var (
	// ErrUnknownMethod is returned when an unknown method is called
	ErrUnknownMethod = errors.New("unknown method")
	// ErrInvalidRequest is returned when an invalid request is made
	ErrInvalidRequest = errors.New("invalid request body")
	// ErrUnprocessableEntity is returned when an entity cannot be processed
	ErrUnprocessableEntity = errors.New("unprocessable entity")
)

func makeSuccessResponse(statusCode statusCode, data any) *ipcResponse {
	return &ipcResponse{
		ConnType:   serverResponseType,
		StatusCode: statusCode,
		Data:       data,
	}
}

func makeErrorResponse(statusCode statusCode, errorMsg string) *ipcResponse {
	return &ipcResponse{
		ConnType:   serverResponseType,
		StatusCode: statusCode,
		Error:      errorMsg,
	}
}

func makeError(statusCode statusCode, err error) *ipcResponse {
	return makeErrorResponse(statusCode, err.Error())
}

func handleHandlersError(err error) *ipcResponse {
	switch {
	case errors.Is(err, ErrUnknownMethod):
		return makeError(statusNotFound, err)
	case errors.Is(err, ErrInvalidRequest):
		return makeError(statusInvalidRequest, err)
	case errors.Is(err, ErrUnprocessableEntity):
		return makeError(statusUnprocessableEntity, err)
	default:
		return makeError(statusInternalError, err)
	}
}

func handleConnection(conn net.Conn, server *locatorServer) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	errChan := make(chan error, 2)

	go func() {
		for {
			if err := listenMessage(conn, reader, server); err != nil {
				errChan <- err
				return
			}
		}
	}()

	go func() {
		for {
			if err := sendMessage(writer, server.requestChan); err != nil {
				errChan <- err
				return
			}
		}
	}()

	err := <-errChan
	if err != nil {
		errorLogger.Printf("Connection closed due to error: %v", err)
	}
}

func listenMessage(conn net.Conn, reader *bufio.Reader, server *locatorServer) error {
	message, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("Error reading message: %v", err)
	}
	infoLogger.Println("Received message: ", message)

	if err := server.handleIncomingMessage(message, conn); err != nil {
		return fmt.Errorf("Error handling message: %v", err)
	}
	return nil
}

func sendMessage(writer *bufio.Writer, broadcastChan chan any) error {
	for {
		message := <-broadcastChan
		if err := writeJson(writer, message); err != nil {
			return err
		}
	}
}

func writeJson(writer *bufio.Writer, message any) error {
	jsonBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("Failed to marshal message: %v", err)

	}

	writeLen, err := writer.Write(append(jsonBytes, '\n'))
	if err != nil {
		return fmt.Errorf("Failed to write message: %v", err)
	}
	if writeLen != (len(jsonBytes) + 1) {
		return fmt.Errorf("Failed to write message: short write")
	}

	err = writer.Flush()
	if err != nil {
		return fmt.Errorf("Failed to flush writer: %v", err)
	}
	infoLogger.Println("Sent message: ", message)

	return nil
}
