// nolint
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/vertexcover-io/locatr"
)

func main() {
	reRankClient := locatr.NewCohereClient(os.Getenv("COHERE_API_KEY"))

	options := locatr.BaseLocatrOptions{
		UseCache:     false,
		ReRankClient: reRankClient,
	} // llm client is created by default by reading the environment variables.
	connectionOpts := locatr.CdpConnectionOptions{
		Port:   9222,
		PageId: "177AE4272FC8BBE48190C697A27942DA",
	}
	connection, err := locatr.CreateCdpConnection(connectionOpts)
	if err != nil {
		fmt.Println(err)
		return
	}
	playWrightLocatr, err := locatr.NewCdpLocatr(connection, options)
	if err != nil {
		fmt.Println(err)
		return
	}

	newsItem, err := playWrightLocatr.GetLocatr("First news link")
	fmt.Println(newsItem)
	if err != nil {
		log.Fatalf("could not get locator: %v", err)
	}
	time.Sleep(5 * time.Second)
}
