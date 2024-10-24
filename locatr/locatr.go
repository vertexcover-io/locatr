package locatr

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

type currentStateDto struct {
	LocatrName string   `json:"locatr_name"`
	Locatrs    []string `json:"locatrs"`
}

type BaseLocatr struct {
	plugin       PluginInterface
	llmClient    LlmClient
	options      BaseLocatrOptions
	currentState map[string][]currentStateDto
	initilized   bool
}

type BaseLocatrOptions struct {
	CachePath string
}

func NewBaseLocatr(plugin PluginInterface, llmClient LlmClient, options BaseLocatrOptions) *BaseLocatr {
	if len(options.CachePath) == 0 {
		options.CachePath = DEFAULT_CACHE_PATH
	}
	return &BaseLocatr{
		plugin:       plugin,
		llmClient:    llmClient,
		options:      options,
		currentState: make(map[string][]currentStateDto),
		initilized:   false,
	}
}
func (l *BaseLocatr) getCurrentUrl() string {
	return l.plugin.EvaluateJsFunction("window.location.href")
}

func (al *BaseLocatr) locateElementId(htmlDOM string, userReq string) (string, error) {
	systemPrompt := LOCATE_ELEMENT_PROMPT
	jsonData, err := json.Marshal(&llmWebInputDto{
		HtmlDom: htmlDOM,
		UserReq: userReq,
	})
	if err != nil {
		return "", err
	}

	prompt := fmt.Sprintf("%s%s", string(systemPrompt), string(jsonData))

	llmResponse, err := al.llmClient.ChatCompletion(prompt)
	if err != nil {
		return "", err
	}

	llmLocatorOutput := &llmLocatorOutputDto{}
	if err = json.Unmarshal([]byte(llmResponse), llmLocatorOutput); err != nil {
		return "", err
	}

	return llmLocatorOutput.LocatorID, nil
}

func (l *BaseLocatr) addLocatrToState(url string, locatrName string, locatrs []string) {
	fmt.Println("Locatrs are", locatrs)
	if _, ok := l.currentState[url]; !ok {
		l.currentState[url] = []currentStateDto{}
	}
	found := false
	for i, v := range l.currentState[url] {
		if v.LocatrName == locatrName {
			l.currentState[url][i].Locatrs = append(l.currentState[url][i].Locatrs, locatrs...)
			return
		}
	}
	if !found {
		l.currentState[url] = append(l.currentState[url], currentStateDto{LocatrName: locatrName, Locatrs: locatrs})
	}
}

func (l *BaseLocatr) initilizeState() {
	if l.initilized {
		return
	}
	err := l.loadLocatorsCache(l.options.CachePath)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("Cache loaded successfully")
	fmt.Println(l.currentState)
	l.initilized = true
}

func (l *BaseLocatr) GetLocatorStr(userReq string) (string, error) {
	if err := l.plugin.EvaluateJsScript(HTML_MINIFIER_JS_CONTENTT); err != nil {
		return "", ErrUnableToLoadJsScripts
	}
	l.initilizeState()
	log.Println("Searching for locator in cache")
	currnetUrl := l.getCurrentUrl()
	locators, err := l.getLocatrsFromState(currnetUrl)

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
	l.addLocatrToState(currnetUrl, userReq, locators)
	value, err := json.Marshal(l.currentState)
	if err != nil {
		return "", err
	}
	if err = writeLocatorsToCache(l.options.CachePath, value); err != nil {
		log.Println(err)
	}
	return validLocators, nil

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

	if err != nil {
		return fmt.Errorf("failed to write record: %v", err)
	}

	return nil
}

func (l *BaseLocatr) loadLocatorsCache(cachePath string) error {
	file, err := os.Open(cachePath)
	if err != nil {
		return fmt.Errorf("failed to open cache file (%s): %v", cachePath, err)
	}
	defer file.Close()
	byteValue, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read cache file (%s): %v", cachePath, err)
	}
	err = json.Unmarshal(byteValue, &l.currentState)
	if err != nil {
		return fmt.Errorf("failed to unmarshal cache file (%s): %v", cachePath, err)
	}
	return nil
}

func (l *BaseLocatr) getLocatrsFromState(key string) ([]string, error) {
	if locatrs, ok := l.currentState[l.getCurrentUrl()]; ok {
		for _, v := range locatrs {
			if v.LocatrName == key {
				return v.Locatrs, nil
			}
		}
	}
	return nil, errors.New("key not found")
}

func (l *BaseLocatr) getMinifiedDomAndLocatorMap() (*ElementSpec, *IdToLocatorMap, error) {
	result := l.plugin.EvaluateJsFunction("minifyHTML()")
	elementSpec := &ElementSpec{}
	if err := json.Unmarshal([]byte(result), elementSpec); err != nil {
		return nil, nil, err
	}

	result = l.plugin.EvaluateJsFunction("mapElementsToJson()")
	idLocatorMap := &IdToLocatorMap{}
	if err := json.Unmarshal([]byte(result), idLocatorMap); err != nil {
		return nil, nil, err
	}

	return elementSpec, idLocatorMap, nil
}

func (l *BaseLocatr) getValidLocator(locators []string) (string, error) {
	for _, locator := range locators {
		if l.plugin.EvaluateJsFunction(fmt.Sprintf("isValidLocator('%s')", locator)) == "true" {
			return locator, nil
		}
	}
	return "", ErrUnableToFindValidLocator
}
