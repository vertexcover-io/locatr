// nolint
package main

/*
Example on how to use locatr with playwright to interact with docker hub.
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
	if _, err := page.Goto("https://hub.docker.com/"); err != nil {
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

	searchBarLocator, err := locatr.GetLocatr("Search Docker Hub input field")
	if err != nil {
		log.Fatalf("could not get locator: %v", err)
	}
	fmt.Println(searchBarLocator.InnerHTML())
	stringToSend := "Python"
	err = searchBarLocator.Fill(stringToSend)
	if err != nil {
		log.Fatalf("could not fill search bar: %v", err)
	}
	searchBarLocator.Press("Enter")
	time.Sleep(5 * time.Second)
	pythonLink, err := locatr.GetLocatr("Link to python repo on docker hub. It has the following description: 'Python is an interpreted, interactive, object-oriented, open-source programming language.'")
	if err != nil {
		log.Fatalf("could not get locator: %v", err)
	}
	log.Println("Clicking on python link")
	err = pythonLink.First().Click()
	if err != nil {
		log.Fatalf("could not click on python link: %v", err)
	}
	time.Sleep(3 * time.Second)
	tagsLocator, err := locatr.GetLocatr("Tags tab on the page. It is made up of anchor tag")
	if err != nil {
		log.Fatalf("could not get locator: %v", err)
	}
	log.Println("Clicking on tags locator")
	err = tagsLocator.Nth(2).Click()
	if err != nil {
		log.Fatalf("could not click on tags locator: %v", err)
	}
	time.Sleep(3 * time.Second)
}