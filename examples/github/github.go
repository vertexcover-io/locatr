// nolint
package main

/*
Example on how to use locatr with playwright to interact with github.
*/
import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/vertexcover-io/locatr/core"
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
	if _, err := page.Goto("https://github.com/vertexcover-io/locatr"); err != nil {
		log.Fatalf("could not navigate to docker hub: %v", err)
	}
	time.Sleep(5 * time.Second) // wait for page to load

	llmClient, err := core.NewLlmClient(
		os.Getenv("LLM_PROVIDER"), // (openai | anthropic),
		os.Getenv("LLM_MODEL_NAME"),
		os.Getenv("LLM_API_KEY"),
	)
	if err != nil {
		log.Fatalf("could not create llm client: %v", err)
	}
	options := core.BaseLocatrOptions{UseCache: true}

	locatr := core.NewPlaywrightLocatr(page, llmClient, options)

	codeDropDownLocatr, err := locatr.GetLocatr("<> Code dropdown")
	if err != nil {
		log.Fatalf("could not get locator: %v", err)
		return
	}
	if err := codeDropDownLocatr.First().Click(); err != nil {
		log.Fatalf("could not click on code dropdown: %v", err)
		return
	}
	downloadZipLocatr, err := locatr.GetLocatr("Download ZIP button on the opened dropdown")
	fmt.Println(downloadZipLocatr.First().InnerHTML())
	time.Sleep(5 * time.Second)
}
