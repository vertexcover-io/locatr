package locatr

import (
	"log"

	"github.com/playwright-community/playwright-go"
	"github.com/vertexcover-io/locatr/golang/constants"
	"github.com/vertexcover-io/locatr/golang/llm"
	playwrightPlugin "github.com/vertexcover-io/locatr/golang/plugins/playwright"
	seleniumPlugin "github.com/vertexcover-io/locatr/golang/plugins/selenium"
	"github.com/vertexcover-io/locatr/golang/reranker"
	"github.com/vertexcover-io/locatr/golang/types"
	"github.com/vertexcover-io/selenium"
)

// Options configures the behavior of the Locatr instance.
type Options struct {
	// LLMClient handles interactions with the Language Learning Model
	LLMClient *llm.LLMClient
	// ReRanker provides document re-ranking capabilities
	ReRanker types.ReRankerInterface
	// CachePath specifies the file location for persisting locator cache
	CachePath string
	// UseCache enables caching of locator results
	UseCache bool
}

// Locatr is the main orchestrator for finding UI elements based on natural language descriptions.
// It combines browser automation, LLM-based element identification, and caching capabilities.
type Locatr struct {
	plugin  types.PluginInterface
	options *Options
	cache   map[string][]types.CacheEntry
}

// NewLocatr creates a new Locatr instance with the specified plugin and options.
// If options are not provided, it will create default LLM client and reranker from environment variables.
// Parameters:
//   - plugin: Browser automation interface (Selenium or Playwright)
//   - options: Configuration options for the Locatr instance
//
// Returns the initialized Locatr instance and any error that occurred during setup.
func NewLocatr(plugin types.PluginInterface, options *Options) (*Locatr, error) {
	if options == nil {
		options = &Options{}
	}
	if options.LLMClient == nil {
		log.Println("LLM client not provided, creating from env")
		llmClient, err := llm.CreateLLMClientFromEnv()
		if err != nil {
			return nil, err
		}
		options.LLMClient = llmClient
	}

	if options.ReRanker == nil {
		log.Println("Reranker not provided, creating from env")
		reRanker, err := reranker.CreateCohereClientFromEnv()
		if err != nil {
			return nil, err
		}
		options.ReRanker = reRanker
	}

	instance := &Locatr{
		plugin:  plugin,
		options: options,
		cache:   make(map[string][]types.CacheEntry),
	}

	if instance.options.UseCache {
		if options.CachePath == "" {
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
func NewPlaywrightLocatr(page *playwright.Page, options *Options) (*Locatr, error) {
	plugin, err := playwrightPlugin.New(page)
	if err != nil {
		return nil, err
	}
	return NewLocatr(plugin, options)
}

// NewSeleniumLocatr creates a Locatr instance configured to use Selenium for browser automation.
// Parameters:
//   - driver: Initialized Selenium WebDriver instance
//   - options: Configuration options for the Locatr instance
//
// Returns the initialized Locatr instance and any error that occurred during setup.
func NewSeleniumLocatr(driver *selenium.WebDriver, options *Options) (*Locatr, error) {
	plugin, err := seleniumPlugin.New(driver)
	if err != nil {
		return nil, err
	}
	return NewLocatr(plugin, options)
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
