# Locatr 
[![Go Reference](https://pkg.go.dev/badge/github.com/vertexcover-io/locatr.svg)](https://pkg.go.dev/github.com/vertexcover-io/locatr)
![Test](https://github.com/vertexcover-io/locatr/actions/workflows/test.yaml/badge.svg)

Locatr package helps you to find HTML locators on a webpage using prompts and llms.

## Overview 
- LLM based HTML locator finder.
- Supports playwright and there will be addon in the future for other web automation engines.  
- Uses cache to reduce calls to llm apis.


Example: 

```go
starButtonLocator, err := locatr.GetLocatr("Star button on the page")
starButtonLocator.click()
```

### Install Locatr with

```
go get github.com/vertexcover-io/locatr
```

## Table of Contents

- [ Quick Example ](#quick-example)
- [ Locatr Settings ](#locatr-options)
- [ LLM Client ](#llm-client)
- [ Cache Schema & Management ](#cache)
- [ Contributing ](#contributing)

### Quick Example

Here's a quick example on how to use the project:

```go
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/vertexcover-io/locatr"
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
		os.Getenv("LLM_PROVIDER"), // (openai | anthropic),
		os.Getenv("LLM_MODEL_NAME"),
		os.Getenv("LLM_API_KEY"),
	)
	if err != nil {
		log.Fatalf("could not create llm client: %v", err)
	}
	options := locatr.BaseLocatrOptions{UseCache: true}

	playWrightlocatr := locatr.NewPlaywrightLocatr(page, llmClient, options)

	searchBarLocator, err := playWrightlocatr.GetLocatr("Search Docker Hub input field")
	if err != nil {
		log.Fatalf("could not get locator: %v", err)
	}
	fmt.Println(searchBarLocator.InnerHTML())
}
```

**Please check the examples directory for more examples.**

#### Locatr Options
`locatr.BaseLocatrOptions` is a struct with two fields used to configure caching in `locatr`.

**Fields**

- **CachePath** (`string`): 
    - Path where the cache will be saved. 
    - Example: `"/path/to/cache/file"`
  
- **UseCache** (`bool`): 
    - Default is `false`. Set to `true` to use the cache.

#### LLM Client

The `LlmClient` is a wrapper around the llm provider you want to use. Supported providers are `anthropic` and `openai`.

```go
import (
	"github.com/vertexcover-io/locatr.
	"os"
)

llmClient, err := locatr.NewLlmClient(
	os.Getenv("LLM_PROVIDER"), // Supported providers: "openai" | "anthropic"
	os.Getenv("LLM_MODEL_NAME"),
	os.Getenv("LLM_API_KEY"),
)
```

### Locatrs

Locatrs are  are wrapper around the main plugin (playwright, selenium). Currently only playwright is supported.

#### PlaywrightLocatr
Create an instance of `PlayWrightLocatr` using 

```go
playWrightLocatr := locatr.NewPlaywrightLocatr(page, llmClient, options)
```

### Methods

- **GetLocatr**: Locates an element using a descriptive string and returns a `Locator` object.
  
  ```go
  searchBarLocator, err := playWrightLocatr.GetLocatr("Search Docker Hub input field")
  ```

### Cache

#### Cache Schema

The cache is stored in JSON format. The schema is as follows:

```json
{
	"Page Full Url" : [
		{
			"locatr_name": "The description of the element you gave",
			"locatrs": [
				"input#search"
			]
		}
	]
}
```

#### Cache Management

To remove the cache, delete the file at the path specified in `BaseLocatrOptions`'s `CachePath`.


### Contributing

We welcome contributions! Please read our [CONTRIBUTING.md](CONTRIBUTING.md) guide to get started.
