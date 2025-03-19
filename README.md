# Locatr

Find locators for UI elements on web pages using natural language.

## Usage

Initialize an automation plugin of your choice.

- Playwright
	<details>
	<summary>Set up a Playwright page</summary>

	```go
	import (
		"log"

		"github.com/playwright-community/playwright-go"
	)

	pw, err := playwright.Run()
	if err != nil {
		log.Fatalf("could not start Playwright: %v", err)
	}
	
	// --- Launch a browser ---
	browser, err := pw.Chromium.Launch(
		playwright.BrowserTypeLaunchOptions{Headless: playwright.Bool(false)},
	)
	// OR, --- Connect to a browser over CDP ---
	browser, err := pw.Chromium.ConnectOverCDP("<cdp-session-url>")

	if err != nil {
		log.Fatalf("could not connect to browser: %v", err)
	}

	browserContext, err := browser.NewContext(
		playwright.BrowserNewContextOptions{BypassCSP: playwright.Bool(true)},
	)
	if err != nil {
		log.Fatalf("could not create browser context: %v", err)
	}

	page, err := browserContext.NewPage()
	if err != nil {
		log.Fatalf("could not create new page: %v", err)
	}

	if _, err := page.Goto("https://github.com/vertexcover-io/locatr"); err != nil {
		log.Fatalf("failed to load URL: %v", err)
	}
	```
	</details>
	
	```go
	import "github.com/vertexcover-io/locatr/golang/plugins"

	plugin := plugins.NewPlaywrightPlugin(&page)
	```

- Selenium
	<details>
	<summary>Set up a Selenium driver</summary>

	```go
	import (
		"log"

		"github.com/vertexcover-io/selenium"
	)
	service, err := selenium.NewChromeDriverService(
		"path/to/chromedriver-executable", 4444,
	)
	if err != nil {
		log.Fatalf("failed to create service: %v", err)
	}

	driver, err := selenium.NewRemote(selenium.Capabilities{}, "")
	// OR, --- Connect to a remote driver session---
	driver, err := selenium.ConnectRemote("<url>", "<session-id>")

	if err != nil {
		log.Fatalf("could not connect to driver: %v", err)
	}

	if err := driver.Get("https://github.com/vertexcover-io/locatr"); err != nil {
		log.Fatalf("failed to load URL: %v", err)
	}
	```
	</details>

	```go
	import "github.com/vertexcover-io/locatr/golang/plugins"

	plugin := plugins.NewSeleniumPlugin(&driver)
	```

Create a Locatr instance.

```go
import locatr "github.com/vertexcover-io/locatr/golang"

locatr, err := locatr.NewLocatr(plugin)
if err != nil {
	log.Fatalf("failed to create locatr: %v", err)
}
```

Locate an element.

```go
completion, err := locatr.Locate("Star button")
if err != nil {
	log.Fatalf("failed to locate element: %v", err)
}
fmt.Println(completion.Locators[0])
```

Calculate the cost of the completion.

```go
costPer1MInputTokens := 3.0
costPer1MOutputTokens := 15.0
cost := completion.CalculateCost(costPer1MInputTokens, costPer1MOutputTokens)
fmt.Printf("Cost: %v\n", cost)
```

Highlight the locator.

```go
imageBytes, err := locatr.Highlight(completion.Locators[0])
if err != nil {
	log.Fatalf("failed to highlight element: %v", err)
}

// Write the image to a file
os.WriteFile("highlighted.png", imageBytes, 0644)
```
