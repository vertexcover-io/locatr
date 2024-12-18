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
		Port:   9222,
		PageId: "177AE4272FC8BBE48190C697A27942DA", // page id can be found by hitting route: http://localhost:9222/json.
	}
	connection, err := locatr.CreateCdpConnection(connectionOpts)
	if err != nil {
		fmt.Println(err)
		return
	}
	// close the cdp connection
	defer connection.Close()

	playWrightLocatr, err := locatr.NewCdpLocatr(connection, options)
	if err != nil {
		log.Fatal(err)
		return
	}

	newsItem, err := playWrightLocatr.GetLocatr("First news link")
	fmt.Println(newsItem)
	if err != nil {
		log.Fatalf("could not get locator: %v", err)
		return
	}
}
