// nolint
package main

/*
Example on how to use locatr without passing the llm client.
*/

import (
	"log"
	"os"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/vertexcover-io/locatr/golang/baseLocatr"
	"github.com/vertexcover-io/locatr/golang/playwrightLocatr"
	"github.com/vertexcover-io/locatr/golang/reranker"
)

func main() {
	pw, err := playwright.Run()
	if err != nil {
		log.Fatalf("could not start playwright: %v", err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(
		playwright.BrowserTypeLaunchOptions{
			Headless: playwright.Bool(false),
		},
	)
	if err != nil {
		log.Fatalf("could not launch browser: %v", err)
	}
	defer browser.Close()

	page, err := browser.NewPage()
	if err != nil {
		log.Fatalf("could not create page: %v", err)
	}
	if _, err := page.Goto("https://news.ycombinator.com/"); err != nil {
		log.Fatalf("could not navigate to new.ycombinator: %v", err)
	}
	time.Sleep(5 * time.Second) // wait for page to load
	reRankClient := reranker.NewCohereClient(os.Getenv("COHERE_API_KEY"))

	options := baseLocatr.BaseLocatrOptions{
		UseCache:     true,
		ReRankClient: reRankClient,
	} // llm client is created by default by reading the environment variables.

	playWrightLocatr := playwrightLocatr.NewPlaywrightLocatr(page, options)

	newsItem, err := playWrightLocatr.GetLocatr("First news link")
	if err != nil {
		log.Fatalf("could not get locator: %v", err)
	}
	newsItem.First().Click()
	time.Sleep(5 * time.Second)
}
