package locatr

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

//go:embed meta/htmlMinifier.js
var HTML_MINIFIER_JS_CONTENTT string

//go:embed meta/locate_element.prompt
var LOCATE_ELEMENT_PROMPT string

var DEFAULT_CACHE_PATH = ".locatr.cache"

// Default file to write locatr results
var DEFAULT_LOCATR_RESULTS_PATH = "locatr_results.json"

var (
	ErrUnableToLoadJsScripts       = errors.New("unable to load JS scripts")
	ErrUnableToMinifyHtmlDom       = errors.New("unable to minify HTML DOM")
	ErrUnableToExtractIdLocatorMap = errors.New("unable to extract ID locator map")
	ErrUnableToLocateElementId     = errors.New("unable to locate element ID")
	ErrInvalidElementIdGenerated   = errors.New("invalid element ID generated")
	ErrUnableToFindValidLocator    = errors.New("unable to find valid locator")
	ErrFailedToWriteCache          = errors.New("failed to write cache")
	ErrFailedToMarshalJson         = errors.New("failed to marshal json")
)

type IdToLocatorMap map[string][]string

type llmLocatorOutputDto struct {
	LocatorID          string `json:"locator_id"`
	completionResponse chatCompletionResponse
}

type locatrResult struct {
	LocatrDescription       string `json:"locatr_description"`
	CacheHit                bool   `json:"cache_hit"`
	Locatr                  string `json:"locatr"`
	InputTokens             int    `json:"input_tokens"`
	OutputTokens            int    `json:"output_tokens"`
	TotalTokens             int    `json:"total_tokens"`
	ChatCompletionTimeTaken int    `json:"time_taken"`
}

type cachedLocatrsDto struct {
	LocatrName string   `json:"locatr_name"`
	Locatrs    []string `json:"locatrs"`
}

type BaseLocatr struct {
	plugin        PluginInterface
	llmClient     LlmClientInterface
	options       BaseLocatrOptions
	cachedLocatrs map[string][]cachedLocatrsDto
	initialized   bool
	logger        logInterface
	locatrResults []locatrResult
}

// BaseLocatrOptions is a struct that holds all the options for the locatr package
type BaseLocatrOptions struct {
	// CachePath is the path to the cache file
	CachePath string
	// UseCache is a flag to enable/disable cache
	UseCache bool
	// LogConfig is the log configuration
	LogConfig LogConfig

	// LocatrResultsFilePath is the path to the file where the locatr results will be written
	// If not provided, the results will be written to DEFAULT_LOCATR_RESULTS_FILE
	ResultsFilePath string
}

// NewBaseLocatr creates a new instance of BaseLocatr
// plugin: (playwright, puppeteer, etc)
// llmClient: struct that are returned by NewLlmClient
// options: All the options for the locatr package
func NewBaseLocatr(plugin PluginInterface, llmClient LlmClientInterface, options BaseLocatrOptions) *BaseLocatr {
	if len(options.CachePath) == 0 {
		options.CachePath = DEFAULT_CACHE_PATH
	}
	if len(options.ResultsFilePath) == 0 {
		options.ResultsFilePath = DEFAULT_LOCATR_RESULTS_PATH
	}
	return &BaseLocatr{
		plugin:        plugin,
		llmClient:     llmClient,
		options:       options,
		cachedLocatrs: make(map[string][]cachedLocatrsDto),
		initialized:   false,
		logger:        NewLogger(options.LogConfig),
		locatrResults: []locatrResult{},
	}
}

func (l *BaseLocatr) addCachedLocatrs(url string, locatrName string, locatrs []string) {
	if _, ok := l.cachedLocatrs[url]; !ok {
		l.logger.Debug(fmt.Sprintf("Domain %s not found in cache... Creating new cachedLocatrsDto", url))
		l.cachedLocatrs[url] = []cachedLocatrsDto{}
	}
	found := false
	for i, v := range l.cachedLocatrs[url] {
		if v.LocatrName == locatrName {
			l.logger.Debug(fmt.Sprintf("Found locatr %s in cache... Updating locators", locatrName))
			l.cachedLocatrs[url][i].Locatrs = GetUniqueStringArray(append(l.cachedLocatrs[url][i].Locatrs, locatrs...))
			return
		}
	}
	if !found {
		l.logger.Debug(fmt.Sprintf("Locatr %s not found in cache... Creating new locatr", locatrName))
		l.cachedLocatrs[url] = append(l.cachedLocatrs[url], cachedLocatrsDto{LocatrName: locatrName, Locatrs: locatrs})
	}
}

func (l *BaseLocatr) initializeState() {
	if l.initialized || !l.options.UseCache {
		l.logger.Debug("Cache disabled or already initialized")
		return
	}
	err := l.loadLocatorsCache(l.options.CachePath)
	if err != nil {
		l.logger.Error(fmt.Sprintf("Failed to load cache: %v", err))
		return
	}
	l.logger.Debug("Cache loaded successfully")
	l.initialized = true
}
func (l *BaseLocatr) getLocatrsFromState(key string, currentUrl string) ([]string, error) {
	if locatrs, ok := l.cachedLocatrs[currentUrl]; ok {
		for _, v := range locatrs {
			if v.LocatrName == key {
				l.logger.Debug(fmt.Sprintf("Key %s found in cache", key))
				return v.Locatrs, nil
			}
		}
	}
	l.logger.Debug(fmt.Sprintf("Key %s not found in cache", key))
	return nil, errors.New("key not found")
}
func (l *BaseLocatr) loadLocatorsCache(cachePath string) error {
	file, err := os.Open(cachePath)
	if err != nil {
		l.logger.Debug(fmt.Sprintf("Cache file not found: %v", err))
		return nil // ignore this error for now
	}
	defer file.Close()
	byteValue, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read cache file (%s): %v", cachePath, err)
	}
	err = json.Unmarshal(byteValue, &l.cachedLocatrs)
	if err != nil {
		return fmt.Errorf("failed to unmarshal cache file (%s): %v", cachePath, err)
	}
	return nil
}
func writeLocatorsToCache(cachePath string, cacheString []byte) error {
	err := os.MkdirAll(filepath.Dir(cachePath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	file, err := os.OpenFile(cachePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()
	if _, err := file.Write(cacheString); err != nil {
		return fmt.Errorf("failed to write cache: %v", err)
	}

	return nil
}

// getLocatorStr returns the locator string for the given user request
func (l *BaseLocatr) getLocatorStr(userReq string) (string, error) {
	if err := l.plugin.evaluateJsScript(HTML_MINIFIER_JS_CONTENTT); err != nil {
		return "", ErrUnableToLoadJsScripts
	}
	l.initializeState()
	l.logger.Info(fmt.Sprintf("Getting locator for user request: %s", userReq))
	currentUrl := l.getCurrentUrl()
	locators, err := l.getLocatrsFromState(userReq, currentUrl)

	if err != nil {
		l.logger.Error(fmt.Sprintf("Failed to get locators from cache: %v", err))
	} else {
		if len(locators) > 0 {
			validLocator, err := l.getValidLocator(locators)
			if err == nil {
				result := locatrResult{
					LocatrDescription: userReq,
					CacheHit:          true,
					Locatr:            validLocator,
				}
				l.locatrResults = append(l.locatrResults, result)
				l.logger.Info(fmt.Sprintf("Cache hit, key: %s, value: %s", userReq, validLocator))
				return validLocator, nil
			} else {
				l.logger.Error(fmt.Sprintf("Failed to find valid locator in cache: %v", err))
			}
			l.logger.Info("All cached locators are outdated.")
		}

	}

	l.logger.Info("Cache miss, starting dom minification")
	minifiedDOM, locatorsMap, err := l.getMinifiedDomAndLocatorMap()
	if err != nil {
		l.logger.Error(fmt.Sprintf("Failed to minify DOM and extract ID locator map: %v", err))
		return "", ErrUnableToMinifyHtmlDom
	}

	l.logger.Info("Extracting element ID using LLM")
	llmOtuput, err := l.locateElementId(minifiedDOM.ContentStr(), userReq)
	if err != nil {
		l.logger.Error(fmt.Sprintf("Failed to locate element ID: %v", err))
		return "", ErrUnableToLocateElementId
	}

	locators, ok := (*locatorsMap)[llmOtuput.LocatorID]
	if !ok {
		l.logger.Error("Invalid element ID generated")
		return "", ErrInvalidElementIdGenerated
	}

	validLocator, err := l.getValidLocator(locators)
	if err != nil {
		l.logger.Error(fmt.Sprintf("Failed to find valid locator: %v", err))
		return "", ErrUnableToFindValidLocator
	}
	if l.options.UseCache {
		l.logger.Info(fmt.Sprintf("Adding locatrs of %s to cache", userReq))
		l.logger.Debug(fmt.Sprintf("Adding Locars of %s: %v to cache", userReq, locators))
		l.addCachedLocatrs(currentUrl, userReq, locators)
		value, err := json.Marshal(l.cachedLocatrs)
		if err != nil {
			l.logger.Error(fmt.Sprintf("Failed to marshal cache: %v", err))
			return "", fmt.Errorf("%w: %v", ErrFailedToMarshalJson, err)
		}
		if err = writeLocatorsToCache(l.options.CachePath, value); err != nil {
			l.logger.Error(fmt.Sprintf("Failed to write cache: %v", err))
			return "", fmt.Errorf("%w: %v", ErrFailedToWriteCache, err)
		}
	}
	result := locatrResult{
		LocatrDescription:       userReq,
		CacheHit:                false,
		Locatr:                  validLocator,
		InputTokens:             llmOtuput.completionResponse.InputTokens,
		OutputTokens:            llmOtuput.completionResponse.OutputTokens,
		TotalTokens:             llmOtuput.completionResponse.TotalTokens,
		ChatCompletionTimeTaken: llmOtuput.completionResponse.TimeTaken,
	}

	l.locatrResults = append(l.locatrResults, result)
	return validLocator, nil

}
func (l *BaseLocatr) getCurrentUrl() string {
	if value, err := l.plugin.evaluateJsFunction("window.location.href"); err == nil {
		return value
	} else {
		l.logger.Error(fmt.Sprintf("Failed to get current URL: %v", err))
	}
	return ""
}

func (l *BaseLocatr) getMinifiedDomAndLocatorMap() (*ElementSpec, *IdToLocatorMap, error) {
	result, _ := l.plugin.evaluateJsFunction("minifyHTML()")
	elementSpec := &ElementSpec{}
	if err := json.Unmarshal([]byte(result), elementSpec); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal ElementSpec json: %v", err)
	}

	result, _ = l.plugin.evaluateJsFunction("mapElementsToJson()")
	idLocatorMap := &IdToLocatorMap{}
	if err := json.Unmarshal([]byte(result), idLocatorMap); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal IdToLocatorMap json: %v", err)
	}

	return elementSpec, idLocatorMap, nil
}

func (l *BaseLocatr) getValidLocator(locators []string) (string, error) {
	for _, locator := range locators {
		if value, _ := l.plugin.evaluateJsFunction(fmt.Sprintf("isValidLocator('%s')", locator)); value == "true" {
			return locator, nil
		}
	}
	return "", ErrUnableToFindValidLocator
}
func (al *BaseLocatr) locateElementId(htmlDOM string, userReq string) (*llmLocatorOutputDto, error) {
	systemPrompt := LOCATE_ELEMENT_PROMPT
	jsonData, err := json.Marshal(&llmWebInputDto{
		HtmlDom: htmlDOM,
		UserReq: userReq,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal llmWebInputDto json: %v", err)
	}

	prompt := fmt.Sprintf("%s%s", string(systemPrompt), string(jsonData))

	llmResponse, err := al.llmClient.ChatCompletion(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to get response from LLM: %v", err)
	}

	llmLocatorOutput := &llmLocatorOutputDto{
		completionResponse: *llmResponse,
	}
	if err = json.Unmarshal([]byte(llmResponse.Completion), llmLocatorOutput); err != nil {
		return nil, fmt.Errorf("failed to unmarshal llmLocatorOutputDto json: %v", err)
	}

	return llmLocatorOutput, nil
}

func (l *BaseLocatr) writeLocatrResultsToFile() {
	file, err := os.OpenFile(l.options.ResultsFilePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		l.logger.Error(fmt.Sprintf("Failed to create file locatr results file: %v", err))
		return
	}
	defer file.Close()
	value, err := json.Marshal(l.locatrResults)
	if err != nil {
		l.logger.Error(fmt.Sprintf("Failed to marshal locatr results json: %v", err))
		return
	}
	if _, err := file.Write(value); err != nil {
		l.logger.Error(fmt.Sprintf("Failed to write locatr results to file: %v", err))
	} else {
		l.logger.Info(fmt.Sprintf("Results written to file: %s", l.options.ResultsFilePath))
	}
}

func (l *BaseLocatr) getLocatrResults() []locatrResult {
	return l.locatrResults
}
