// nolint
package main

/*
Example on how to use locatr with playwright to interact with steam.
*/

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/playwright-community/playwright-go"
	locatr "github.com/vertexcover-io/locatr/golang"
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
			Args:     []string{"--disable-blink-features=AutomationControlled"},
			IgnoreDefaultArgs: []string{
				"--enable-automation",
			},
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
	if _, err := page.Goto("https://store.steampowered.com/"); err != nil {
		log.Fatalf("could not navigate to steam store: %v", err)
	}
	time.Sleep(5 * time.Second) // wait for page to load

	llmClient, err := locatr.NewLlmClient(
		locatr.OpenAI, // (openai | anthropic),
		os.Getenv("LLM_MODEL_NAME"),
		os.Getenv("LLM_API_KEY"),
	)
	if err != nil {
		log.Fatalf("could not create llm client: %v", err)
	}
	options := locatr.BaseLocatrOptions{UseCache: true, LogConfig: locatr.LogConfig{Level: locatr.Silent}, LlmClient: llmClient}

	playWrightLocatr := locatr.NewPlaywrightLocatr(page, options)

	searchBarLocator, err := playWrightLocatr.GetLocatr("Search input bar on the steam store.")
	if err != nil {
		log.Fatalf("could not get search bar locator: %v", err)
	}
	if err := searchBarLocator.First().Fill("Counter Strike 2"); err != nil {
		log.Fatalf("could not fill search bar: %v", err)
		return
	}
	if err := searchBarLocator.First().Press("Enter"); err != nil {
		log.Fatalf("could not press enter: %v", err)
		return
	}
	time.Sleep(5 * time.Second)
	counterStrike2Locator, err := playWrightLocatr.GetLocatr("Counter Strike 2 game on he list")
	if err != nil {
		log.Fatalf("could not get first video locator: %v", err)
		return
	}
	if err := counterStrike2Locator.First().Click(); err != nil {
		log.Fatalf("could not click first video: %v", err)
		return
	}
	time.Sleep(5 * time.Second)
	systemRequirementsLocator, err := playWrightLocatr.GetLocatr("System Requirements section on the game page.")
	if err != nil {
		log.Fatalf("could not get system requirements locator: %v", err)
		return
	}
	fmt.Println(systemRequirementsLocator.First().InnerHTML())
}
