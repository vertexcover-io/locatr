package main

import (
	"flag"
	"log"

	"github.com/vertexcover-io/locatr/ipc"
)

func main() {
	port := flag.Int("port", 50051, "The port to start the tcp server on")
	flag.Parse()

	if err := ipc.StartTCPServer(*port); err != nil {
		log.Fatalf("Failed to start tcp server: %v", err)
	}
}
