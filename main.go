package main

import (
	"fmt"
	"log"
	"os"

	"github.com/playwright-community/playwright-go"
	locatr "github.com/vertexcover-io/locatr/golang"
)

func main() {
	os.Setenv("LLM_PROVIDER", "anthropic")
	os.Setenv("LLM_MODEL", "claude-3-5-sonnet-latest")
	os.Setenv("LLM_API_KEY", "<anthropic-api-key>")
	os.Setenv("COHERE_API_KEY", "<cohere-api-key>")

	pw, err := playwright.Run()
	if err != nil {
		log.Fatalf("could not start Playwright: %v", err)
	}
	defer pw.Stop()

	// Launch browser in headful mode
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false),
	})
	if err != nil {
		log.Fatalf("could not launch browser: %v", err)
	}
	defer browser.Close()

	browserContext, err := browser.NewContext(playwright.BrowserNewContextOptions{
		BypassCSP: playwright.Bool(true),
	})
	if err != nil {
		log.Fatalf("could not create browser context: %v", err)
	}

	page, err := browserContext.NewPage()
	if err != nil {
		log.Fatalf("could not create new page: %v", err)
	}

	url := "https://github.com"
	if _, err := page.Goto(url); err != nil {
		log.Fatalf("failed to load URL: %v", err)
	}

	locatr, err := locatr.NewPlaywrightLocatr(&page, &locatr.Options{UseCache: true})
	if err != nil {
		log.Fatalf("could not create playwright locatr: %v", err)
	}

	query := "Button to subscribe to newsletter"
	useGrounding := true

	completion, err := locatr.Locate(query, useGrounding)
	fmt.Println(completion.LLMCompletionMeta)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("First locator:", completion.Locators[0])
}
