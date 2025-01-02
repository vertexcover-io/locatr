// nolint
package main

/*
Example on how to use locatr with playwright to interact with docker hub.
*/

import (
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
	if _, err := page.Goto("https://hub.docker.com/"); err != nil {
		log.Fatalf("could not navigate to docker hub: %v", err)
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
	options := locatr.BaseLocatrOptions{UseCache: true, LogConfig: locatr.LogConfig{Level: locatr.Debug}, LlmClient: llmClient}

	playWrightLocatr := locatr.NewPlaywrightLocatr(page, options)

	searchBarLocator, err := playWrightLocatr.GetLocatr("Search Docker Hub input field")
	if err != nil {
		log.Fatalf("could not get locator: %v", err)
	}
	stringToSend := "Python"
	err = searchBarLocator.Fill(stringToSend)
	if err != nil {
		log.Fatalf("could not fill search bar: %v", err)
	}
	searchBarLocator.Press("Enter")
	time.Sleep(5 * time.Second)
	pythonLink, err := playWrightLocatr.GetLocatr("Link to python repo on docker hub. It has the following description: 'Python is an interpreted, interactive, object-oriented, open-source programming language.'")
	if err != nil {
		log.Fatalf("could not get locator: %v", err)
	}
	log.Println("Clicking on python link")
	err = pythonLink.First().Click()
	if err != nil {
		log.Fatalf("could not click on python link: %v", err)
	}
	time.Sleep(3 * time.Second)
	tagsLocator, err := playWrightLocatr.GetLocatr("Tags tab on the page. It is made up of anchor tag")
	if err != nil {
		log.Fatalf("could not get locator: %v", err)
	}
	log.Println("Clicking on tags locator")
	err = tagsLocator.Nth(2).Click()
	if err != nil {
		log.Fatalf("could not click on tags locator: %v", err)
	}
	playWrightLocatr.WriteResultsToFile()
	time.Sleep(3 * time.Second)
}
