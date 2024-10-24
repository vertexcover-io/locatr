// nolint

/*
This is an example of how to use the locatr package to interact with a webpage using Playwright and an LLM (Large Language Model) client.
In this example, we launch a Chromium browser, navigate to Docker Hub, interact with the search input field by filling in a search term ('Python'),
and perform additional actions like clicking on a Python repository link and navigating to its 'Tags' tab. The locatr package is used for web element
identification, and an LLM client is integrated for contextual descriptions of the locators. Predefined locators are leveraged to simplify interaction
with the webpage elements.
*/
package main

import (
	"fmt"
	_ "fmt"
	"log"
	"os"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/vertexcover-io/locatr/core"
	"github.com/vertexcover-io/locatr/llm"
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

	llmClient, err := llm.NewLlmClient(
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
