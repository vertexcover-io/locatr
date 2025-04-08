// nolint
package main

/*
Example on how to use locatr with playwright to interact with docker hub.
*/

import (
	"log"
	"time"

	"github.com/playwright-community/playwright-go"
	locatr "github.com/vertexcover-io/locatr/golang"
	"github.com/vertexcover-io/locatr/golang/plugins"
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
	if err != nil {
		log.Fatalf("could not create llm client: %v", err)
	}

	plugin, err := plugins.NewPlaywrightPlugin(&page)
	if err != nil {
		log.Fatalf("could not create playwright plugin: %v", err)
	}
	locatr, err := locatr.NewLocatr(plugin)
	sBarCompletion, err := locatr.Locate("Search Docker Hub input field")
	if err != nil {
		log.Fatalf("could not get locator: %v", err)
	}
	stringToSend := "Python"
	err = page.Locator(sBarCompletion.Locators[0]).Fill(stringToSend)
	if err != nil {
		log.Fatalf("could not fill search bar: %v", err)
	}
	err = page.Locator(sBarCompletion.Locators[0]).Press("Enter")
	if err != nil {
		log.Fatalf("could not press enter: %v", err)
	}
	time.Sleep(5 * time.Second)

	pLink, err := locatr.Locate("Link to python repo on docker hub. It has the following description: 'Python is an interpreted, interactive, object-oriented, open-source programming language.'")
	if err != nil {
		log.Fatalf("could not get locator: %v", err)
	}
	log.Println("Clicking on python link")
	err = page.Locator(pLink.Locators[0]).Click()
	if err != nil {
		log.Fatalf("could not click on python link: %v", err)
	}
	time.Sleep(3 * time.Second)

	tagsLoc, err := locatr.Locate("Tags tab on the page. It is made up of anchor tag")
	if err != nil {
		log.Fatalf("could not get locator: %v", err)
	}
	log.Println("Clicking on tags locator")
	err = page.Locator(tagsLoc.Locators[0]).Nth(2).Click()
	if err != nil {
		log.Fatalf("could not click on tags locator: %v", err)
	}
	time.Sleep(3 * time.Second)
}
