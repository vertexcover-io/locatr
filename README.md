# Locatr 

LLM based html element locator using natural language. 


## Table of Contents

- [ Quick Example ](#quick-example)

- [ Locatr Settings ](#locatr-options)


#### Locatr Options
`core.BaseLocatrOptions` is a struct with two fields used to configure caching in `locatr`.
**Fields**

- **CachePath** (`string`): 
    - Path where the cache will be saved. 
    - Example: `"/path/to/cache/file"`
  
- **UseCache** (`bool`): 
    - Default is `false`. Set to `true` to use the cache.



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
}
```

## 
