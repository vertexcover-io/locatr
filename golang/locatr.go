package locatr

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"

	"github.com/playwright-community/playwright-go"
	"github.com/vertexcover-io/locatr/golang/constants"
	"github.com/vertexcover-io/locatr/golang/llm"
	"github.com/vertexcover-io/locatr/golang/options"
	playwrightPlugin "github.com/vertexcover-io/locatr/golang/plugins/playwright"
	seleniumPlugin "github.com/vertexcover-io/locatr/golang/plugins/selenium"
	"github.com/vertexcover-io/locatr/golang/reranker"
	"github.com/vertexcover-io/locatr/golang/types"
	"github.com/vertexcover-io/locatr/golang/utils"
	"github.com/vertexcover-io/selenium"
)

// Locatr is the main orchestrator for finding UI elements based on natural language descriptions.
// It combines browser automation, LLM-based element identification, and caching capabilities.
type Locatr struct {
	plugin  types.PluginInterface
	options *options.LocatrOptions
	cache   map[string][]types.CacheEntry
}

// NewLocatr creates a new Locatr instance with the specified plugin and options.
// If options are not provided, it will create default LLM client and reranker from environment variables.
// Parameters:
//   - plugin: Browser automation interface (Selenium or Playwright)
//   - options: Configuration options for the Locatr instance
//
// Returns the initialized Locatr instance and any error that occurred during setup.
func NewLocatr(plugin types.PluginInterface, opts *options.LocatrOptions) (*Locatr, error) {
	if opts == nil {
		opts = &options.LocatrOptions{}
	}
	if opts.LLMClient == nil {
		log.Println("LLM client not provided, creating from env")
		llmClient, err := llm.CreateLLMClientFromEnv()
		if err != nil {
			return nil, err
		}
		opts.LLMClient = llmClient
	}

	if opts.ReRanker == nil {
		log.Println("Reranker not provided, creating from env")
		reRanker, err := reranker.CreateCohereClientFromEnv()
		if err != nil {
			return nil, err
		}
		opts.ReRanker = reRanker
	}

	instance := &Locatr{
		plugin:  plugin,
		options: opts,
		cache:   make(map[string][]types.CacheEntry),
	}

	if instance.options.UseCache {
		if opts.CachePath == "" {
			log.Printf("Cache path not provided, using %v\n", constants.DEFAULT_CACHE_PATH)
			instance.options.CachePath = constants.DEFAULT_CACHE_PATH
		}
		log.Printf("Loading cache from %v\n", instance.options.CachePath)
		if err := instance.loadCache(); err != nil {
			return nil, err
		}
	}
	return instance, nil
}

// NewPlaywrightLocatr creates a Locatr instance configured to use Playwright for browser automation.
// Parameters:
//   - page: Initialized Playwright page instance
//   - options: Configuration options for the Locatr instance
//
// Returns the initialized Locatr instance and any error that occurred during setup.
func NewPlaywrightLocatr(page *playwright.Page, options *options.LocatrOptions) (*Locatr, error) {
	return NewLocatr(playwrightPlugin.New(page), options)
}

// NewSeleniumLocatr creates a Locatr instance configured to use Selenium for browser automation.
// Parameters:
//   - driver: Initialized Selenium WebDriver instance
//   - options: Configuration options for the Locatr instance
//
// Returns the initialized Locatr instance and any error that occurred during setup.
func NewSeleniumLocatr(driver *selenium.WebDriver, options *options.LocatrOptions) (*Locatr, error) {
	return NewLocatr(seleniumPlugin.New(driver), options)
}

// Locate finds UI elements matching the provided natural language description.
// Parameters:
//   - userRequest: Natural language description of the element to find
//   - useGrounding: When true, uses visual grounding (screenshots) for element location
//
// Returns:
//   - LocatrCompletion containing found locators and metadata
//   - error if element location fails
//
// The function follows this process:
//  1. Checks cache if enabled
//  2. Gets current DOM state
//  3. Processes request using either ID-based or visual grounding approach
//  4. Updates cache if enabled
func (l *Locatr) Locate(userRequest string, useGrounding bool) (types.LocatrCompletion, error) {
	completion := &types.LocatrCompletion{
		Locators:    []string{},
		LocatorType: "",
		CacheHit:    false,
		LLMCompletionMeta: types.LLMCompletionMeta{
			TimeTaken:    0,
			InputTokens:  0,
			OutputTokens: 0,
			TotalTokens:  0,
			Provider:     l.options.LLMClient.GetProvider(),
			Model:        l.options.LLMClient.GetModel(),
		},
	}

	if err := l.plugin.WaitForLoadEvent(nil); err != nil {
		return *completion, err
	}

	url := l.plugin.GetCurrentContext()
	log.Printf("['%s'] | Processing request '%s'\n", url, userRequest)

	if l.options.UseCache {
		if err := l.processCacheRequest(completion, userRequest); err == nil {
			return *completion, nil
		}
	}

	dom, err := l.plugin.GetMinifiedDOM()
	if err != nil {
		return *completion, err
	}

	if !useGrounding {
		err = l.processIdRequest(completion, userRequest, dom)
	} else {
		err = l.processPointRequest(completion, userRequest, dom)
	}
	if len(completion.Locators) == 0 || err != nil {
		return *completion, err
	}

	if l.options.UseCache {
		if _, ok := l.cache[url]; !ok {
			l.cache[url] = []types.CacheEntry{}
		}

		l.cache[url] = append(l.cache[url], types.CacheEntry{
			UserRequest: userRequest,
			Locators:    completion.Locators,
			LocatorType: dom.Metadata.LocatorType,
		})

		return *completion, l.persistCache()
	}
	return *completion, nil
}

// Highlight takes a locator and returns a screenshot of the element with a highlight overlay.
// Parameters:
//   - locator: The locator of the element to highlight
//   - opts: Configuration options for the highlight
//
// Returns:
//   - screenshot: The screenshot of the element with a highlight overlay in PNG bytes format
//   - error if element location fails
func (l *Locatr) Highlight(locator string, opts *options.HighlightOptions) ([]byte, error) {
	if err := l.plugin.SetViewportSize(1280, 800); err != nil {
		return nil, err
	}

	location, err := l.plugin.GetLocation(locator)
	if err != nil {
		return nil, err
	}

	screenshot, err := l.plugin.TakeScreenshot()
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(screenshot))
	if err != nil {
		return nil, err
	}
	// Convert to RGBA for drawing
	rgbaImg := image.NewRGBA(img.Bounds())
	draw.Draw(rgbaImg, img.Bounds(), img, image.Point{}, draw.Src)

	if opts == nil {
		opts = &options.HighlightOptions{}
	}
	if opts.Color == nil {
		opts.Color = &color.RGBA{255, 0, 0, 255}
	}
	if opts.Radius == 0 {
		opts.Radius = 10
	}
	if opts.Opacity == 0 {
		opts.Opacity = 0.5
	}
	utils.HighlightPoint(&location.Point, rgbaImg, opts)

	buf := new(bytes.Buffer)
	if err := png.Encode(buf, rgbaImg); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
