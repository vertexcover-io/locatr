# Locatr 

Locatr is a package which will help your scraping DX (Developer Experience). It supports
multiple automation web-drivers using plugins. You're able to generate the locator using
a simple `locator_description` in natural language.

## Getting Started

### Example Usage

Here's a quick example of how to use the project:

```go
package main

import (
    "log"
    "os"

    "github.com/playwright-community/playwright-go"
    "github.com/vertexcover-io/locatr/llm"
    "github.com/vertexcover-io/locatr/plugins"
)

func main() {
    pw, err := playwright.Run()
    if err != nil {
	    log.Fatalf("could not start playwright: %v", err)
    }
    defer pw.Stop()

    browser, err := pw.Chromium.Launch()
    if err != nil {
	    log.Fatalf("could not launch browser: %v", err)
    }
    defer browser.Close()

    page, err := browser.NewPage()
    if err != nil {
	    log.Fatalf("could not create page: %v", err)
    }
    if _, err := page.Goto("https://www.youtube.com/"); err != nil {
        log.Fatalf("could not navigate to YouTube: %v", err)
    }

    llmClient, err := llm.NewLlmClient(
        os.Getenv("LLM_PROVIDER"), // (openai | anthropic),
        os.Getenv("LLM_MODEL_NAME"),
        os.Getenv("LLM_API_KEY"), 
    )
    if err != nil {
        log.Fatalf("could not create llm client: %v", err)
    }

    locatr := plugins.NewPlaywrightLocatr(page, llmClient)

    searchBarLocator, err := locatr.GetLocatr("search bar")
    if err != nil {
        log.Fatalf("could not get locator: %v", err)
    }
    if err = searchBarLocator.Click(); err != nil {
        log.Fatalf("could not click search bar: %v", err)
    }
}
```

## Future Plans

- **Multi Language Support**: Compile the core `BaseLocatr` to **wasm** and use **wasmer** for using `Locatr` in multiple languages.
- **Locator Caching**: Cache the locators so that llm won't be called when running the scraper everytime.

