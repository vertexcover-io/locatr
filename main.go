package main

import (
	"fmt"
	"image/color"
	"log"
	"os"

	"github.com/playwright-community/playwright-go"
	locatr "github.com/vertexcover-io/locatr/golang"
	"github.com/vertexcover-io/locatr/golang/options"
	"github.com/vertexcover-io/selenium"
	"github.com/vertexcover-io/selenium/chrome"
)

func testPlaywright(url, query string, useGrounding, useCache bool) {
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

	if _, err := page.Goto(url); err != nil {
		log.Fatalf("failed to load URL: %v", err)
	}

	locatr, err := locatr.NewPlaywrightLocatr(&page, &options.LocatrOptions{UseCache: useCache})
	if err != nil {
		log.Fatalf("could not create playwright locatr: %v", err)
	}

	completion, err := locatr.Locate(query, useGrounding)
	fmt.Println(completion.LLMCompletionMeta)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("First locator:", completion.Locators[0])
	highlight, err := locatr.Highlight(completion.Locators[0], &options.HighlightOptions{
		Color:   &color.RGBA{255, 0, 0, 255},
		Radius:  10,
		Opacity: 0.5,
	})
	if err != nil {
		log.Fatal(err)
	}
	err = os.WriteFile("playwright-highlight.png", highlight, 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func testPlaywrightOverCDP(url, query string, useGrounding, useCache bool) {
	pw, err := playwright.Run()
	if err != nil {
		log.Fatalf("could not start Playwright: %v", err)
	}
	defer pw.Stop()

	// Launch browser in headful mode
	_, err = pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false),
		Args:     []string{"--remote-debugging-port=9222"},
	})
	if err != nil {
		log.Fatalf("could not launch browser: %v", err)
	}

	// Connect to browser over CDP

	browser, err := pw.Chromium.ConnectOverCDP("http://localhost:9222")
	if err != nil {
		log.Fatalf("could not connect to browser: %v", err)
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

	if _, err := page.Goto(url); err != nil {
		log.Fatalf("failed to load URL: %v", err)
	}

	locatr, err := locatr.NewPlaywrightLocatr(&page, &options.LocatrOptions{UseCache: useCache})
	if err != nil {
		log.Fatalf("could not create playwright locatr: %v", err)
	}

	completion, err := locatr.Locate(query, useGrounding)
	fmt.Println(completion.LLMCompletionMeta)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("First locator:", completion.Locators[0])
	highlight, err := locatr.Highlight(completion.Locators[0], &options.HighlightOptions{
		Color:   &color.RGBA{255, 0, 0, 255},
		Radius:  10,
		Opacity: 0.5,
	})
	if err != nil {
		log.Fatal(err)
	}
	err = os.WriteFile("playwright-cdp-highlight.png", highlight, 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func testSelenium(url, query string, useGrounding, useCache bool) {

	service, err := selenium.NewChromeDriverService(
		"F:\\locatr-refac\\driver\\chromedriver-win64\\chromedriver.exe",
		4444,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer service.Stop()

	caps := selenium.Capabilities{}
	caps.AddChrome(chrome.Capabilities{Args: []string{}})

	driver, err := selenium.NewRemote(caps, "")
	if err != nil {
		log.Fatal(err)
	}
	defer driver.Close()

	driver.Get(url)
	locatr, err := locatr.NewSeleniumLocatr(&driver, &options.LocatrOptions{UseCache: useCache})
	if err != nil {
		log.Fatalf("could not create selenium locatr: %v", err)
	}

	completion, err := locatr.Locate(query, useGrounding)
	fmt.Println(completion.LLMCompletionMeta)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("First locator:", completion.Locators[0])
	highlight, err := locatr.Highlight(completion.Locators[0], &options.HighlightOptions{
		Color:   &color.RGBA{255, 0, 0, 255},
		Radius:  10,
		Opacity: 0.5,
	})
	if err != nil {
		log.Fatal(err)
	}
	err = os.WriteFile("selenium-highlight.png", highlight, 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	os.Setenv("LLM_PROVIDER", "anthropic")
	os.Setenv("LLM_MODEL", "claude-3-5-sonnet-latest")
	os.Setenv("LLM_API_KEY", "<your-anthropic-api-key>")
	os.Setenv("COHERE_API_KEY", "<your-cohere-api-key>")

	url := "https://makemytrip.com"
	query := "Button to close the login popup"
	useCache := false

	// testPlaywright(url, query, false, useCache)
	testPlaywrightOverCDP(url, query, true, useCache)
	// testSelenium(url, query, false, useCache)
}
