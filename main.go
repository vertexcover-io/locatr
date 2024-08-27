// package main
//
// import (
// 	"flag"
// 	"log"
//
// 	"github.com/vertexcover-io/locatr/rpc"
// )

// func main() {
// 	port := flag.Int("port", 50051, "The port to start the gRPC server on")
// 	flag.Parse()
//
// 	if err := rpc.StartGRPCServer(*port); err != nil {
// 		log.Fatalf("Failed to start gRPC server: %v", err)
// 	}
// }

package main

import (
	"log"
	"os"

	"github.com/playwright-community/playwright-go"
	"github.com/vertexcover-io/locatr/locatr"
	"github.com/vertexcover-io/locatr/plugins"
)

func main() {
	pw, err := playwright.Run()
	if err != nil {
		log.Fatalf("could not start playwright: %v", err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{Headless: playwright.Bool(false)})
	if err != nil {
		log.Fatalf("could not launch browser: %v", err)
	}
	defer browser.Close()

	page, err := browser.NewPage()
	if err != nil {
		log.Fatalf("could not create page: %v", err)
	}
	if _, err := page.Goto("https://www.youtube.com/"); err != nil {
		log.Fatalf("could not navigate to YouTube: %v", err)
	}

	llmConfig := locatr.LlmConfig{
		ApiKey:   os.Getenv("LLM_API_KEY"),
		Provider: os.Getenv("LLM_PROVIDER"), // (openai | anthropic),
		Model:    os.Getenv("LLM_MODEL_NAME"),
	}
	if err != nil {
		log.Fatalf("could not create llm client: %v", err)
	}

	locatr, err := plugins.NewPlaywrightLocatr(page, &locatr.LocatrConfig{LlmConfig: llmConfig, CachePath: ".locactr.cache"})
	if err != nil {
		log.Fatalf("could not create locatr: %v", err)
	}

	searchBarLocator, err := locatr.GetLocatr("search bar")
	if err != nil {
		log.Fatalf("could not get locator: %v", err)
	}
	if err = searchBarLocator.Click(); err != nil {
		log.Fatalf("could not click search bar: %v", err)
	}

	if err = searchBarLocator.Fill("3b1b"); err != nil {
		log.Fatalf("could not fill search bar: %v", err)
	}

	searchButtonLocator, err := locatr.GetLocatr("search button")
	if err != nil {
		log.Fatalf("could not get locator: %v", err)
	}
	if err = searchButtonLocator.Click(); err != nil {
		log.Fatalf("could not click search button: %v", err)
	}

	page.WaitForTimeout(10_000)
}
