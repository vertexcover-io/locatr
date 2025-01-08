# Locatr 
[![Go Reference](https://pkg.go.dev/badge/github.com/vertexcover-io/locatr.svg)](https://pkg.go.dev/github.com/vertexcover-io/locatr)
![Test](https://github.com/vertexcover-io/locatr/actions/workflows/test.yaml/badge.svg)

Locatr package helps you to find HTML locators on a webpage using prompts and llms. 

## Overview 
- LLM based HTML element path finder.
- Re-rank support for improved accuracy.
- Supports playwright, selenium, cdp, appium.  
- Uses cache to reduce calls to llm apis.
- Results/Statistics generation of api calls.

Example: 

```go
starButtonLocator, err := locatr.GetLocatr("Star button on the page")
starButtonLocator.click()
```

### Install Locatr with

#### Golang

```
go get github.com/vertexcover-io/locatr/golang
```

#### Python
```
pip install locatr
```

## Table of Contents

- [ Quick Example ](#quick-example)
- [ LLM Client ](#llm-client)
- [ Re-ranking Client ](#re-ranking-client)
- [ Locatr Settings ](#locatr-options)
- [ Locatrs ](#locatrs)
- [ Cache Schema & Management ](#cache)
- [ Logging ](#logging)
- [ Generate Statistics ](#locatr-results)
- [ Contributing ](#contributing)

### Quick Example

#### Python example

```
# example assumes that there is already a page opened in the selenium session.
import os

from locatr import (
    LlmProvider,
    LlmSettings,
    Locatr,
    LocatrCdpSettings,
    LocatrSeleniumSettings,
    PluginType,
)

llm_settings = LlmSettings(
    llm_provider=LlmProvider.OPENAI,
    llm_api_key=os.environ.get("LLM_API_KEY"),
    model_name=os.environ.get("LLM_MODEL_NAME"),
    reranker_api_key=os.environ.get("RERANKER_API_KEY"),
)

locatr_settings_selenium = LocatrSeleniumSettings(
    llm_settings=llm_settings,
    selenium_url=os.environ.get("SELENIUM_URL"), # url must end with `/wd/hub`
    selenium_session_id="e4c543363b9000a66073db7a39152719",
)

selenium_locatr = Locatr(locatr_settings_selenium, debug=True)

print(selenium_locatr.get_locatr("H1 element with text Example Domain"))

```
For more examples check the `examples/python` folder.

Find the python documentation [here](python_client/README.md).

#### Go example

```go
package main

import (
	"fmt"
	"github.com/vertexcover-io/locatr/golang/baseLocatr"
	"github.com/vertexcover-io/locatr/golang/reranker"
	"github.com/vertexcover-io/locatr/golang/seleniumLocatr"
	"github.com/vertexcover-io/selenium"
	"github.com/vertexcover-io/selenium/chrome"
	"log"
	"os"
	"time"
)

func main() {
	service, err := selenium.NewChromeDriverService("./chromedriver-linux64/chromedriver", 4444)
	if err != nil {
		log.Fatal(err)
		return
	}
	caps := selenium.Capabilities{}
	caps.AddChrome(chrome.Capabilities{Args: []string{}})

	defer service.Stop()
	driver, err := selenium.NewRemote(caps, "")
	if err != nil {
		log.Fatal(err)
		return
	}
	driver.Get("https://news.ycombinator.com/")

	reRankClient := reranker.NewCohereClient(os.Getenv("COHERE_API_KEY"))

	options := baseLocatr.BaseLocatrOptions{
		UseCache:     true,
		ReRankClient: reRankClient,
	} 

	// wait for page to load
	time.Sleep(3 * time.Second)

	seleniumLocatr, err := seleniumLocatr.NewRemoteConnSeleniumLocatr(
		"http://localhost:4444/wd/hub", "ca0d56a6a3dcfc51eb0110750f0abab7", options) // the path must end with /wd/hub

	if err != nil {
		log.Fatal(err)
		return
	}
	newsLocatr, err := seleniumLocatr.GetLocatrStr("First news link in the site..")
	if err != nil {
		log.Fatal(err)
		return
	}
	fmt.Println(newsLocatr)
}

```

**Please check the examples directory for more examples.**

### LLM Client
The `LlmClient` is a wrapper around the llm provider you want to use. Supported providers are `locatr.OpenAI`, `locatr.Anthropic`. It is optional; if not provided in the options, Locatr will automatically create an LlmClient using environment variables. 

- The following environment variables will be read to create a default LlmClient:
	- **LLM_PROVIDER**: Defines which provider's LLM should be utilized (`openai`, `anthropic`).
	- **LLM_MODEL**: Specifies the model to use 
	- **LLM_API_KEY**: The API key required to authenticate with the LLM provider.

To create a new llm client call the `locatr.NewLlmClient` function.

```go
import (
	"github.com/vertexcover-io/locatr/golang/llm"
	"github.com/vertexcover-io/locatr/golang/baseLocatr"
	"os"
)

llmClient, err := llm.NewLlmClient(
	llm.OpenAI, // Supported providers: "openai" | "anthropic"
	os.Getenv("LLM_MODEL_NAME"),
	os.Getenv("LLM_API_KEY"),
)
options := baseLocatr.BaseLocatrOptions{
	LlmClient: llmClient,
}
```

Run without creating the llm client..

```go
import (
	"github.com/vertexcover-io/locatr/golang/baseLocatr"
	"os"
)

options := baseLocatr.BaseLocatrOptions{
	UseCache: true,
}
```


### Re-ranking Client

`ReRankClient` is a wrapper around the ranking provider you want to use. Currently, we only support the `cohere` re-ranker. To create a `cohere` re-ranker, use the following code:

- The default `cohere` re-ranking model is `rerank-english-v3.0`.

```go
import (
	"github.com/vertexcover-io/locatr/golang/reranker"
	"github.com/vertexcover-io/locatr/golang/baseLocatr"
	"os"
)

reRankClient, err := reranker.NewCohereClient(
	os.Getenv("COHERE_API_KEY"),
)
options := baseLocatr.BaseLocatrOptions{
	ReRankClient: reRankClient,
}
```

**Advantages of using re-ranking in Locatr**

- Using re-ranking reduces the input context sent to the LLM.
- Re-ranked chunks will contain only the most relevant HTML chunks, improving the accuracy.
- Sending less input context to the LLM reduces response time and lowers the cost per LLM call.

### Locatr Options
`baseLocatr.BaseLocatrOptions` is a struct with multiple fields used to configure caching, logging, and output file paths in `locatr`.

**Fields**

- **CachePath** (`string`): 
    - Path where the cache will be saved. 
    - Example: `"/path/to/cache/file"`
  
- **UseCache** (`bool`): 
    - Default is `false`. Set to `true` to enable caching.

- **LogConfig** (`LogConfig`): 
    - Configuration for logging behavior.
  
    - **Level** (`LogLevel`): 
        - Sets the log level. Controls the verbosity of logging.
        - Example: `locatr.Info` to log errors, warnings, and info messages.

    - **Writer** (`Writer`): 
        - Destination for log output. Implement the `Printf` function for custom log handling.

- **ResultsFilePath** (`string`): 
    - Path to the file where `locatr` results will be saved.
    - If not provided, results will be saved to `DEFAULT_LOCATR_RESULTS_FILE`.

- **LlmClient** (`LlmClientInterface`): 
    - Optional value; if not provided will be created by default ([read more about llm client](#llm-client))

- **ReRankClient** (`ReRankInterface`)
	- The `ReRankClient` you want to use. When this is passed locatr will use the re-ranking client to re-rank the html chunks. ([More about re-ranking](#re-ranking-client)).

### Locatrs

Locatrs are a wrapper around the main plugin (playwright, selenium, cdp).

#### PlaywrightLocatr
Create an instance of `PlayWrightLocatr` using :

```go
playWrightLocatr := playwrightLocatr.NewPlaywrightLocatr(page, llmClient, options)
```

#### CdpLocatr
To use Locatr through CDP, we first need to start the browser with a CDP server. This can be achieved by running:
```
google-chrome --remote-debugging-port=9222
```
We can pass the same arguments when using Selenium or Playwright:

- Selenium:
```
chrome_options = Options()
chrome_options.add_argument("--remote-debugging-port=9222")
```

- Playwright:
```
browser = playwright.chromium.launch(headless=False, args=["--remote-debugging-port=9222"])
```

Then we create cdp connection.

```
connectionOpts := cdpLocatr.CdpConnectionOptions{
	Port:   9222,
}
connection, err := cdpLocatr.CreateCdpConnection(connectionOpts)
defer connection.Close()
```

Now we can create the CDP Locatr with:

```
cdpLocatr, err := cdpLocatr.NewCdpLocatr(connection, options)
```

#### Selenium Locatr

Selenium Locatr can be created through two ways:

1. Through selenium server url:
```
seleniumLocatr, err := seleniumLocatr.NewRemoteConnSeleniumLocatr("http://localhost:4444/wd/hub", driver.SessionID(), options) 
```
**note: the path must always be `/wd/hub`**

2. Directly passing the selenium driver:
```
seleniumLocatr, err := seleniumLocatr.NewSeleniumLocatr(driver, options)
```


### Methods

- **GetLocatr**: Locates an element using a descriptive string and returns a `Locator` object.
  
```go
searchBarLocator, err := cdpLocatr.GetLocatr("Search Docker Hub input field")
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

### Logging 

Logging is enabled by default in locatr and it's set to `Error` log level. Pass the `LogConfig` value in the `BaseLocatrOptions` struct.

```
options := baseLocatr.BaseLocatrOptions{UseCache: true, LogConfig: locatr.LogConfig{Level: locatr.Debug}}
```

#### Available Log Levels

The following log levels are available, in increasing order of priority:

- `Debug`: Logs all messages, info, warn, error.
- `Info` : Logs informational messages, warnings, and errors.
- `Warning`: Logs warnings and errors only.
- `Error` (Default): Logs only error messages.

### Locatr Results

Locatr provides a feature to get all the information about each locatr request made (**call to GetLocatr function**). The result has the following schema.

- **LocatrDescription** (`string`): Description of the locatr passed to the request.
- **Url** (`string`): The URL associated with the locatr.
- **CacheHit** (`bool`): Indicates if the result was retrieved from the cache (`true`) or freshly generated (`false`).
- **Locatr** (`string`): The locatr generated by the operation.
- **InputTokens** (`int`): Number of input tokens processed by the LLM call.
- **OutputTokens** (`int`): Number of tokens generated in the output by the LLM call.
- **TotalTokens** (`int`): Sum of input and output tokens.
- **LlmErrorMessage** (`string`): The error message from the LLM, if any.
- **ChatCompletionTimeTaken** (`int`): Time taken for the LLM to complete locatr generation in seconds.
- **AttemptNo** (`int`): An integer field to indicate the attempt number with re rank.
- **LocatrRequestInitiatedAt** (`time.Time`): The timestamp when the request was initiated.
- **LocatrRequestCompletedAt** (`time.Time`): The timestamp when the request was completed.
- **AllLocatrs** (`[]string`): All the locatrs of each located elements.

**Saving Results**

Results can be saved to a file specified by `baseLocatr.BaseLocatrOptions.ResultsFilePath`. If no file path is specified, results are written to `locatr_results.json`.

- **To write results to a file**: Use the `playwrightLocatr.WriteResultsToFile` function.

Schema of the json file:

```json
{
    "locatr_description": "",
    "url": "",
    "cache_hit": false,
    "locatr": "",
    "input_tokens": 8399,
    "output_tokens": 22,
    "total_tokens": 8421,
    "llm_error_message": "",
    "llm_locatr_generation_time_taken": 1,
    "attempt_no": 0,
    "request_initiated_at": "",
    "request_completed_at": "",
	"all_locatrs": []
}
```
- **To retrieve results as a slice**: Use the `GetLocatrResults` function on the locatr struct.


### Contributing

We welcome contributions! Please read our [CONTRIBUTING.md](CONTRIBUTING.md) guide to get started.
