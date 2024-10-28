package core

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

//go:embed meta/htmlMinifier.js
var HTML_MINIFIER_JS_CONTENTT string

//go:embed meta/locate_element.prompt
var LOCATE_ELEMENT_PROMPT string

var DEFAULT_CACHE_PATH = ".locatr.cache"

var (
	ErrUnableToLoadJsScripts       = errors.New("unable to load JS scripts")
	ErrUnableToMinifyHtmlDom       = errors.New("unable to minify HTML DOM")
	ErrUnableToExtractIdLocatorMap = errors.New("unable to extract ID locator map")
	ErrUnableToLocateElementId     = errors.New("unable to locate element ID")
	ErrInvalidElementIdGenerated   = errors.New("invalid element ID generated")
	ErrUnableToFindValidLocator    = errors.New("unable to find valid locator")
)

type IdToLocatorMap map[string][]string

type llmWebInputDto struct {
	HtmlDom string `json:"html_dom"`
	UserReq string `json:"user_req"`
}

type llmLocatorOutputDto struct {
	LocatorID string `json:"locator_id"`
}

type LlmClient interface {
	ChatCompletion(prompt string) (string, error)
}

type cachedLocatrsDto struct {
	LocatrName string   `json:"locatr_name"`
	Locatrs    []string `json:"locatrs"`
}

type BaseLocatr struct {
	plugin        PluginInterface
	llmClient     LlmClient
	options       BaseLocatrOptions
	cachedLocatrs map[string][]cachedLocatrsDto
	initilized    bool
}

// BaseLocatrOptions is a struct that holds all the options for the locatr package
type BaseLocatrOptions struct {
	// CachePath is the path to the cache file
	CachePath string
	// UseCache is a flag to enable/disable cache
	UseCache bool
}

// NewBaseLocatr creates a new instance of BaseLocatr
// plugin: (playwright, puppeteer, etc)
// llmClient: struct that are returned by NewLlmClient
// options: All the options for the locatr package
func NewBaseLocatr(plugin PluginInterface, llmClient LlmClient, options BaseLocatrOptions) *BaseLocatr {
	if len(options.CachePath) == 0 {
		options.CachePath = DEFAULT_CACHE_PATH
	}
	return &BaseLocatr{
		plugin:        plugin,
		llmClient:     llmClient,
		options:       options,
		cachedLocatrs: make(map[string][]cachedLocatrsDto),
		initilized:    false,
	}
}

func (l *BaseLocatr) addCachedLocatrs(url string, locatrName string, locatrs []string) {
	if _, ok := l.cachedLocatrs[url]; !ok {
		l.cachedLocatrs[url] = []cachedLocatrsDto{}
	}
	found := false
	for i, v := range l.cachedLocatrs[url] {
		if v.LocatrName == locatrName {
			l.cachedLocatrs[url][i].Locatrs = GetUniqueStringArray(append(l.cachedLocatrs[url][i].Locatrs, locatrs...))
			return
		}
	}
	if !found {
		l.cachedLocatrs[url] = append(l.cachedLocatrs[url], cachedLocatrsDto{LocatrName: locatrName, Locatrs: locatrs})
	}
}

func (l *BaseLocatr) initilizeState() {
	if l.initilized || !l.options.UseCache {
		return
	}
	err := l.loadLocatorsCache(l.options.CachePath)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("Cache loaded successfully")
	l.initilized = true
}
func (l *BaseLocatr) getLocatrsFromState(key string, currentUrl string) ([]string, error) {
	if locatrs, ok := l.cachedLocatrs[currentUrl]; ok {
		for _, v := range locatrs {
			if v.LocatrName == key {
				return v.Locatrs, nil
			}
		}
	}
	return nil, errors.New("key not found")
}
func (l *BaseLocatr) loadLocatorsCache(cachePath string) error {
	file, err := os.Open(cachePath)
	if err != nil {
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
	l.initilizeState()
	log.Println("Searching for locator in cache")
	currnetUrl := l.getCurrentUrl()
	locators, err := l.getLocatrsFromState(userReq, currnetUrl)

	if err == nil && len(locators) > 0 {
		validLocator, err := l.getValidLocator(locators)
		if err == nil {
			log.Println("Cache hit; returning locator")
			return validLocator, nil
		}
		log.Println("All cached locators are outdated.")
	}

	log.Println("Cache miss; going forward with dom minification")
	minifiedDOM, locatorsMap, err := l.getMinifiedDomAndLocatorMap()
	if err != nil {
		return "", err
	}

	log.Println("Extracting element ID using LLM")
	elementID, err := l.locateElementId(minifiedDOM.ContentStr(), userReq)
	if err != nil {
		return "", err
	}

	locators, ok := (*locatorsMap)[elementID]
	if !ok {
		return "", ErrInvalidElementIdGenerated
	}

	validLocators, err := l.getValidLocator(locators)
	if err != nil {
		log.Println(err)
		return "", ErrUnableToFindValidLocator
	}
	if l.options.UseCache {
		l.addCachedLocatrs(currnetUrl, userReq, locators)
		value, err := json.Marshal(l.cachedLocatrs)
		if err != nil {
			return "", err
		}
		if err = writeLocatorsToCache(l.options.CachePath, value); err != nil {
			log.Println(err)
		}
	}
	return validLocators, nil

}
func (l *BaseLocatr) getCurrentUrl() string {
	if value, err := l.plugin.evaluateJsFunction("window.location.href"); err == nil {
		return value
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
func (al *BaseLocatr) locateElementId(htmlDOM string, userReq string) (string, error) {
	systemPrompt := LOCATE_ELEMENT_PROMPT
	jsonData, err := json.Marshal(&llmWebInputDto{
		HtmlDom: htmlDOM,
		UserReq: userReq,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal llmWebInputDto json: %v", err)
	}

	prompt := fmt.Sprintf("%s%s", string(systemPrompt), string(jsonData))

	llmResponse, err := al.llmClient.ChatCompletion(prompt)
	if err != nil {
		return "", fmt.Errorf("failed to get response from LLM: %v", err)
	}

	llmLocatorOutput := &llmLocatorOutputDto{}
	if err = json.Unmarshal([]byte(llmResponse), llmLocatorOutput); err != nil {
		return "", fmt.Errorf("failed to unmarshal llmLocatorOutputDto json: %v", err)
	}

	return llmLocatorOutput.LocatorID, nil
}
