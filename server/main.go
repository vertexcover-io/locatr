package main

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
)

func acceptConnection(fd net.Conn) {
	defer fd.Close()
	lengthBuff := make([]byte, 4)
	for {
		_, err := fd.Read(lengthBuff)
		fmt.Println("ran .....")
		msgLength := binary.BigEndian.Uint32(lengthBuff)
		if err != nil {
			log.Fatalf("Failed to read message of length %d: %v", msgLength, err)
			continue
		}
		message := make([]byte, msgLength)
		_, err = fd.Read(message)
		if err != nil {
			log.Println("Error reading actual message", err)
		}

		var clientMessage incomingMessage
		if err := json.Unmarshal(message, &clientMessage); err != nil {
			log.Printf("Error parsing json: %v", err)
			continue
		}
		err = validateIncomingMessage(clientMessage)
		if err != nil {
			fd.Write(dumpJson(
				outgoingMessage{
					Type:     clientMessage.Type,
					Status:   "error",
					ClientId: clientMessage.ClientId,
					Error:    err.Error(),
				}))
			continue
		}
		fd.Write(dumpJson(
			outgoingMessage{
				Type:     clientMessage.Type,
				Status:   "ok",
				ClientId: clientMessage.ClientId,
			}))

	}
}

func main() {
	var socketFilePath string
	flag.StringVar(&socketFilePath, "socketFilePath", "/tmp/locatr.sock", "path to the socketfile to listen at.")
	flag.Parse()

	socket, err := net.Listen("unix", socketFilePath)
	if err != nil {
		log.Fatalf("failed connecting to socket: %v", err)
		return
	}
	fmt.Println("Ready to accept connections")
	for {
		fd, err := socket.Accept()
		if err != nil {
			log.Fatal("Failed accepting socket %w", err)
		}
		go acceptConnection(fd)
	}
}
