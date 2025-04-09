package locatr

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/vertexcover-io/locatr/pkg/internal/constants"
	"github.com/vertexcover-io/locatr/pkg/internal/utils"
	"github.com/vertexcover-io/locatr/pkg/llm"
	"github.com/vertexcover-io/locatr/pkg/logging"
	"github.com/vertexcover-io/locatr/pkg/mode"
	"github.com/vertexcover-io/locatr/pkg/reranker"
	"github.com/vertexcover-io/locatr/pkg/types"
)

// Locatr is the main orchestrator for finding UI elements based on natural language descriptions.
// It combines browser automation, LLM-based element identification, and caching capabilities.
type Locatr struct {
	plugin types.PluginInterface
	config *config
	cache  map[string][]types.CacheEntry
}

// config configures the behavior of the Locatr instance.
type config struct {
	llmClient      types.LLMClientInterface
	rerankerClient types.RerankerClientInterface
	mode           types.LocatrMode
	useCache       bool
	cachePath      string
	logger         *slog.Logger
}

// Option is a function that configures the config.
type Option func(*config)

// WithLLMClient sets the llm for the config.
func WithLLMClient(client types.LLMClientInterface) Option {
	return func(opts *config) {
		opts.llmClient = client
	}
}

// WithRerankerClient sets the reranker for the config.
func WithRerankerClient(client types.RerankerClientInterface) Option {
	return func(opts *config) {
		opts.rerankerClient = client
	}
}

// WithMode sets the mode for the config.
func WithMode(mode types.LocatrMode) Option {
	return func(opts *config) {
		opts.mode = mode
	}
}

// EnableCache enables caching for the config.
// If path is nil, it will use the constants.DEFAULT_CACHE_PATH.
func EnableCache(path *string) Option {
	return func(opts *config) {
		opts.useCache = true
		if path != nil {
			opts.cachePath = *path
		} else {
			opts.cachePath = constants.DEFAULT_CACHE_PATH
		}
	}
}

// WithLogger sets the logger for the config.
func WithLogger(logger *slog.Logger) Option {
	return func(opts *config) {
		opts.logger = logger
	}
}

// NewLocatr creates a new Locatr instance with the specified plugin and options.
// If options are not provided, it will create default LLM client and reranker from environment variables.
// Parameters:
//   - plugin: Browser automation interface (Selenium or Playwright)
//   - opts: Configuration options for the Locatr instance
//
// Returns the initialized Locatr instance and any error that occurred during setup.
func NewLocatr(plugin types.PluginInterface, opts ...Option) (*Locatr, error) {

	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.logger == nil {
		cfg.logger = logging.DefaultLogger
	}

	if cfg.llmClient == nil {
		llmClient, err := llm.DefaultLLMClient(cfg.logger)
		if err != nil {
			return nil, err
		}
		cfg.llmClient = llmClient
	}

	if cfg.rerankerClient == nil {
		rerankerClient, err := reranker.DefaultRerankerClient(cfg.logger)
		if err != nil {
			return nil, err
		}
		cfg.rerankerClient = rerankerClient
	}

	if cfg.mode == nil {
		cfg.mode = &mode.DOMAnalysisMode{}
	}

	instance := &Locatr{
		plugin: plugin,
		config: cfg,
		cache:  make(map[string][]types.CacheEntry),
	}

	if instance.config.useCache {
		cfg.logger.Info("Loading cache", "path", instance.config.cachePath)
		if err := instance.loadCache(); err != nil {
			return nil, err
		}
	}
	return instance, nil
}

// Locate finds UI elements matching the provided natural language description.
// Parameters:
//   - request: Natural language description of the element to find
//
// Returns:
//   - LocatrCompletion containing found locators and metadata
//   - error if element location fails
func (l *Locatr) Locate(request string) (types.LocatrCompletion, error) {
	defer logging.CreateTopic(fmt.Sprintf("[Locate] '%s'", request), l.config.logger)()

	completion := &types.LocatrCompletion{
		Locators:    []string{},
		LocatorType: "",
		CacheHit:    false,
		LLMCompletionMeta: types.LLMCompletionMeta{
			InputTokens:  0,
			OutputTokens: 0,
			Provider:     l.config.llmClient.GetProvider(),
			Model:        l.config.llmClient.GetModel(),
		},
	}

	if l.config.useCache {
		if err := l.processCacheRequest(request, completion); err == nil {
			return *completion, nil
		} else {
			l.config.logger.Error("couldn't process cache request", "error", err)
		}
	}

	err := l.config.mode.ProcessRequest(
		request,
		l.plugin,
		l.config.llmClient,
		l.config.rerankerClient,
		l.config.logger,
		completion,
	)
	if len(completion.Locators) == 0 || err != nil {
		return *completion, err
	}

	if l.config.useCache {
		url, err := l.plugin.GetCurrentContext()
		if err == nil && url != nil {
			if _, ok := l.cache[*url]; !ok {
				l.cache[*url] = []types.CacheEntry{}
			}

			l.cache[*url] = append(l.cache[*url], types.CacheEntry{
				UserRequest: request,
				Locators:    completion.Locators,
				LocatorType: completion.LocatorType,
			})

			if err := l.persistCache(); err != nil {
				l.config.logger.Error("couldn't persist cache", "error", err)
			}

			return *completion, err
		}
	}
	return *completion, nil
}

// Highlight takes a locator and returns a screenshot of the element with a highlight overlay.
// Parameters:
//   - locator: The locator of the element to highlight
//   - config: Configuration for the highlight
//
// Returns:
//   - screenshot: The screenshot of the element with a highlight overlay in PNG bytes format
//   - error if element location fails
func (l *Locatr) Highlight(locator string, config *types.HighlightConfig) ([]byte, error) {

	if config == nil {
		config = &types.HighlightConfig{}
	}
	if config.Color == nil {
		config.Color = &color.RGBA{255, 0, 0, 255}
	}
	if config.Radius == 0 {
		config.Radius = 10
	}
	if config.Opacity == 0 {
		config.Opacity = 0.5
	}
	if config.Resolution == nil {
		config.Resolution = &types.Resolution{Width: 1280, Height: 800}
	}

	if err := l.plugin.SetViewportSize(config.Resolution.Width, config.Resolution.Height); err != nil {
		return nil, err
	}

	location, err := l.plugin.GetElementLocation(locator)
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

	utils.DrawPoint(rgbaImg, &location.Point, config)

	buf := new(bytes.Buffer)
	if err := png.Encode(buf, rgbaImg); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Compare checks if the given locators point to the same UI element on the page.
// It compares the scroll position and point of each locator to determine if they
// refer to the same element.
//
// Parameters:
//   - locators: The locators to compare.
//
// Returns:
//   - bool: True if all locators point to the same element, false otherwise.
//   - error: An error if any locator cannot be resolved or if a comparison fails.
func (l *Locatr) Compare(locators ...string) (bool, error) {
	if len(locators) < 2 {
		return false, errors.New("at least two locators are required for comparison")
	}

	// Get the location of the first locator
	firstLoc, err := l.plugin.GetElementLocation(locators[0])
	if err != nil {
		return false, err
	}

	// Compare the rest of the locators with the first one
	for _, locator := range locators[1:] {
		loc, err := l.plugin.GetElementLocation(locator)
		if err != nil {
			return false, err
		}
		if !firstLoc.ScrollPosition.Equals(loc.ScrollPosition) || !firstLoc.Point.Equals(loc.Point) {
			return false, nil
		}
	}

	return true, nil
}

// loadCache reads and deserializes the cache file into memory.
// Returns nil if the cache file doesn't exist, error if reading or parsing fails.
func (l *Locatr) loadCache() error {
	file, err := os.Open(l.config.cachePath)
	if err != nil {
		// return nil if file doesn't exist. It will be created later by persistCache method
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("couldn't open file: %v", err)
	}
	defer file.Close()

	byteContent, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("couldn't read file: %v", err)
	}
	if len(byteContent) == 0 {
		return nil
	}

	if err := json.Unmarshal(byteContent, &l.cache); err != nil {
		return fmt.Errorf("error loading cache: %v", err)
	}
	return nil
}

// persistCache serializes and writes the current cache to disk.
// Creates necessary directories and handles file permissions.
// Returns error if writing fails.
func (l *Locatr) persistCache() error {
	l.config.logger.Info("Writing cache to disk")
	cacheBytes, err := json.Marshal(l.cache)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(l.config.cachePath), 0755); err != nil {
		return err
	}
	file, err := os.OpenFile(l.config.cachePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	if _, err := file.Write(cacheBytes); err != nil {
		return fmt.Errorf("failed to write cache: %v", err)
	}
	return nil
}

// processCacheRequest attempts to find locators associated with the user request and current context in the cache.
// Parameters:
//   - request: Natural language description to look up
//   - completion: Output structure to populate with cache results
//
// Returns error if no valid cached locators are found.
func (l *Locatr) processCacheRequest(request string, completion *types.LocatrCompletion) error {
	l.config.logger.Info("Searching for locators in cache")
	url, err := l.plugin.GetCurrentContext()
	if err != nil && url == nil {
		return errors.New("couldn't get current context")
	}
	if entries, ok := l.cache[*url]; ok {
		for _, entry := range entries {
			if entry.UserRequest != request {
				continue
			}

			validLocators := []string{}
			for _, locator := range entry.Locators {
				ok, err := l.plugin.IsLocatorValid(locator)
				if err != nil || !ok {
					continue
				}
				validLocators = append(validLocators, locator)
			}

			if len(validLocators) > 0 {
				l.config.logger.Info("Cache hit", "request", entry.UserRequest)
				completion.Locators = validLocators
				completion.LocatorType = entry.LocatorType
				completion.CacheHit = true
				return nil
			}
		}
	}
	return fmt.Errorf("no cache entry found for user request: %v", request)
}
