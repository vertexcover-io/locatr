/*
Locatr simplifies the process of generating HTML element locators from your straightforward descriptions. It currently supports Playwright and will soon offer compatibility with Selenium and other frameworks. With its internal caching system, Locatr enhances efficiency in locator generation, making your web automation tasks smoother.

Example:

	starButtonLocator, err := locatr.GetLocatr("Star button on the page")
	starButtonLocator.click()

To get started, let's open a new page in the browser using playwright-go.

	func main() {
		pw, _ := playwright.Run()
		defer pw.Stop()
		browser, _ := pw.Chromium.Launch(
			playwright.BrowserTypeLaunchOptions{
				Headless: playwright.Bool(false),
			},
		)
		defer browser.Close()
		page, _ := browser.NewPage()
		page.Goto("https://hub.docker.com/")
	}

After opening a page, we need to create a new LLM client. Currently, we support `anthropic` and `openai`. The `LlmClient` struct is available in `github.com/vertexcover-io/locatr/core`.

	import (
		"github.com/vertexcover-io/locatr"
		"os"
	)

	llmClient, err := locatr.NewLlmClient(
		os.Getenv("LLM_PROVIDER"), // Supported providers: "openai" | "anthropic"
		os.Getenv("LLM_MODEL_NAME"),
		os.Getenv("LLM_API_KEY"),
	)

Once we have an `llmClient`, we need to configure `BaseLocatrOptions`. This struct sets caching and configuration options.

	options := locatr.BaseLocatrOptions{UseCache: true, CachePath: ".locatr.cache"}

- `CachePath` has a default value of ".locatr.cache".
- `UseCache` is false by default. To enable caching, set `UseCache` to `true`.

Now, we can create a new `PlaywrightLocatr` instance by calling `NewPlaywrightLocatr` with the `page`, `llmClient`, and `options`.

	playWrightLocatr := locatr.NewPlaywrightLocatr(page, llmClient, options)

To locate an element, use the `GetLocatr` method. This method takes a descriptive string as input and returns a `Locator` object.

	searchBarLocator, err := playWrightLocatr.GetLocatr("Search Docker Hub input field")

The `GetLocatr` method returns either an error or a Playwright locator object. You can then interact with the located element through the returned locator.

	fmt.Println(searchBarLocator.InnerHTML())
	stringToSend := "Python"
	err = searchBarLocator.Fill(stringToSend)

The cache is stored in json format. The schema is as follows:

	{
		"Page Full Url" : [
			{
				"locatr_name": "The description of the element you gave",
				"locatrs": [
					input#search",
				]
			}
		]
	}

To remove the cache, you should delete the file at the path specified in `BaseLocatrOptions`.
*/
package locatr
