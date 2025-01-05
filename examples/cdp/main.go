package main

import (
	"fmt"
	"log"
	"os"

	"github.com/vertexcover-io/locatr/golang/baseLocatr"
	cdpLocatr "github.com/vertexcover-io/locatr/golang/cdp"
	"github.com/vertexcover-io/locatr/golang/reranker"
)

func main() {
	reRankClient := reranker.NewCohereClient(os.Getenv("COHERE_API_KEY"))

	options := baseLocatr.BaseLocatrOptions{
		UseCache:     false,
		ReRankClient: reRankClient,
	} // llm client is created by default by reading the environment variables.

	// connect to the remote cdp server. CDP server can be started by passing `--remote-debugging-port` arg while starting the browser.
	connectionOpts := cdpLocatr.CdpConnectionOptions{
		Port: 9222,
	}
	connection, err := cdpLocatr.CreateCdpConnection(connectionOpts)
	if err != nil {
		fmt.Println(err)
		return
	}
	// close the cdp connection
	defer connection.Close()

	cdpLocatr, err := cdpLocatr.NewCdpLocatr(connection, options)
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
