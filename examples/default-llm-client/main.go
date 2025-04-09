// nolint
package main

/*
Example on how to use locatr without passing the llm client.
*/

import (
	"fmt"
	"log"
	"time"

	"github.com/playwright-community/playwright-go"
	locatr "github.com/vertexcover-io/locatr/pkg"
	"github.com/vertexcover-io/locatr/pkg/plugins"
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

	plugin, err := plugins.NewPlaywrightPlugin(&page)
	if err != nil {
		log.Fatal("failed creating playwright plugin", err)
	}
	locatr, err := locatr.NewLocatr(plugin, locatr.EnableCache(nil))
	if err != nil {
		log.Fatal("failed creating playwright locatr locatr", err)
	}

	completion, err := locatr.Locate("Search Docker Hub input field")
	if err != nil {
		log.Fatalf("could not get locator: %v", err)
	}
	fmt.Println(completion)
}
