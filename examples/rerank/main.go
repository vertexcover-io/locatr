// nolint
package main

/*
Example on how to use locatr without passing the llm client.
*/

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/playwright-community/playwright-go"
	locatr "github.com/vertexcover-io/locatr/golang"
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

	options := locatr.BaseLocatrOptions{
		UseCache:     true,
		ReRankClient: reRankClient,
	} // llm client is created by default by reading the environment variables.

	ctx := context.Background()
	playWrightLocatr := playwrightLocatr.NewPlaywrightLocatr(ctx, page, options)

	nItem, err := playWrightLocatr.GetLocatr(ctx, "First news link")
	if err != nil {
		log.Fatalf("could not get locator: %v", err)
	}
	if err := page.Locator(nItem.Selectors[0]).Click(); err != nil {
		log.Fatalf("could not click news item: %v", err)
	}
	time.Sleep(5 * time.Second)

}
