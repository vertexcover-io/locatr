// nolint
package main

/*
Example on how to use locatr with playwright to interact with github.
*/
import (
	"fmt"
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
	if _, err := page.Goto("https://github.com/vertexcover-io/locatr"); err != nil {
		log.Fatalf("could not navigate to docker hub: %v", err)
	}
	time.Sleep(5 * time.Second) // wait for page to load

	plugin, err := plugins.NewPlaywrightPlugin(&page)
	if err != nil {
		log.Fatalf("could not create playwright plugin: %v", err)
	}
	locatr, err := locatr.NewLocatr(plugin)
	if err != nil {
		log.Fatalf("could not create locatr: %v", err)
	}

	cDropDownCompletion, err := locatr.Locate("<> Code dropdown")
	if err != nil {
		log.Fatalf("could not get locator: %v", err)
		return
	}
	if err := page.Locator(cDropDownCompletion.Locators[0]).Click(); err != nil {
		log.Fatalf("could not click on code dropdown: %v", err)
		return
	}

	dZipCompletion, err := locatr.Locate("Download ZIP button on the opened dropdown")
	if err != nil {
		log.Fatalf("could not get download ZIP locator: %v", err)
		return
	}
	fmt.Println(page.Locator(dZipCompletion.Locators[0]).InnerHTML())
	time.Sleep(5 * time.Second)

}
