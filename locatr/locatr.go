package locatr

import (
	_ "embed"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
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

type BaseLocatr struct {
	plugin    PluginInterface
	llmClient LlmClient
	options   BaseLocatrOptions
}

type BaseLocatrOptions struct {
	CachePath string
}

func NewBaseLocatr(plugin PluginInterface, llmClient LlmClient, options BaseLocatrOptions) *BaseLocatr {
	if len(options.CachePath) == 0 {
		options.CachePath = DEFAULT_CACHE_PATH
	}
	return &BaseLocatr{
		plugin:    plugin,
		llmClient: llmClient,
		options:   options,
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

func (l *BaseLocatr) GetLocatorStr(userReq string) (string, error) {
	if err := l.plugin.EvaluateJsScript(HTML_MINIFIER_JS_CONTENTT); err != nil {
		return "", ErrUnableToLoadJsScripts
	}

	log.Println("Searching for locator in cache")
	hashingKey := fmt.Sprintf("%s-%s", l.getCurrentUrl(), userReq)
	locators, err := readLocatorsFromCache(l.options.CachePath, hashingKey)
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

	validLocator, err := l.getValidLocator(locators)
	if err != nil {
		log.Println(err)
		return "", ErrUnableToFindValidLocator
	}

	if err = writeLocatorsToCache(l.options.CachePath, hashingKey, locators); err != nil {
		log.Println(err)
	}
	return validLocator, nil

}

func writeLocatorsToCache(cachePath string, key string, locators []string) error {
	err := os.MkdirAll(filepath.Dir(cachePath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	file, err := os.OpenFile(cachePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	record := append([]string{key}, locators...)
	err = writer.Write(record)
	if err != nil {
		return fmt.Errorf("failed to write record: %v", err)
	}

	return nil
}

func readLocatorsFromCache(cachePath string, key string) ([]string, error) {
	file, err := os.Open(cachePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %v", err)
	}

	for _, record := range records {
		if len(record) > 0 && record[0] == key {
			return record[1:], nil
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
