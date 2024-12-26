package main

import (
	"fmt"
	"log"
	"os"

	"github.com/vertexcover-io/locatr"
)

func main() {
	reRankClient := locatr.NewCohereClient(os.Getenv("COHERE_API_KEY"))

	options := locatr.BaseLocatrOptions{
		UseCache:     false,
		ReRankClient: reRankClient,
	} // llm client is created by default by reading the environment variables.

	// connect to the remote cdp server. CDP server can be started by passing `--remote-debugging-port` arg while starting the browser.
	connectionOpts := locatr.CdpConnectionOptions{
		Port: 9222,
	}
	connection, err := locatr.CreateCdpConnection(connectionOpts)
	if err != nil {
		fmt.Println(err)
		return
	}
	// close the cdp connection
	defer connection.Close()

	cdpLocatr, err := locatr.NewCdpLocatr(connection, options)
	if err != nil {
		log.Fatal(err)
		return
	}

	newsItem, err := cdpLocatr.GetLocatrStr("First news link")
	fmt.Println(newsItem)
	if err != nil {
		log.Fatalf("could not get locator: %v", err)
		return
	}
}
