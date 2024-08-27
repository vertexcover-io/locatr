package locatr

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/vertexcover-io/locatr/llm"
)

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

type PluginInterface interface {
	EvaluateJs(jsStr string) string
}

type BaseLocatr struct {
	plugin    PluginInterface
	llmClient LlmClient
	cachePath string
}

func NewBaseLocatr(plugin PluginInterface, conf *LocatrConfig) (*BaseLocatr, error) {
	llmClient, err := llm.NewLlmClient(conf.LlmConfig.Provider, conf.LlmConfig.Model, conf.LlmConfig.ApiKey)
	if err != nil {
		return nil, err
	}

	return &BaseLocatr{
		plugin:    plugin,
		llmClient: llmClient,
		cachePath: conf.CachePath,
	}, nil
}

func hashLocatorKeyWithUrl(url string, userReq string) string {
	fullKey := fmt.Sprintf("%s-%s", url, userReq)
	hash := sha256.Sum256([]byte(fullKey))
	return hex.EncodeToString(hash[:])
}

func (l *BaseLocatr) locateElementId(htmlDOM string, userReq string) (string, error) {
	systemPrompt, err := readStaticFile("static/locate_element.prompt")
	if err != nil {
		return "", err
	}

	jsonData, err := json.Marshal(&llmWebInputDto{
		HtmlDom: htmlDOM,
		UserReq: userReq,
	})
	if err != nil {
		return "", err
	}

	prompt := fmt.Sprintf("%s%s", string(systemPrompt), string(jsonData))

	llmResponse, err := l.llmClient.ChatCompletion(prompt)
	if err != nil {
		return "", err
	}

	llmLocatorOutput := &llmLocatorOutputDto{}
	if err = json.Unmarshal([]byte(llmResponse), llmLocatorOutput); err != nil {
		return "", err
	}

	return llmLocatorOutput.LocatorID, nil
}

func (l *BaseLocatr) loadJsScripts() error {
	jsBytes, err := readStaticFile("static/htmlMinifier.js")
	if err != nil {
		return err
	}

	l.plugin.EvaluateJs(string(jsBytes))
	return nil
}

func (l *BaseLocatr) getMinifiedDomAndLocatorMap() (*ElementSpec, *IdToLocatorMap, error) {
	result := l.plugin.EvaluateJs("minifyHTML()")
	elementSpec := &ElementSpec{}
	if err := json.Unmarshal([]byte(result), elementSpec); err != nil {
		return nil, nil, err
	}

	result = l.plugin.EvaluateJs("mapElementsToJson()")
	idLocatorMap := &IdToLocatorMap{}
	if err := json.Unmarshal([]byte(result), idLocatorMap); err != nil {
		return nil, nil, err
	}

	return elementSpec, idLocatorMap, nil
}

func (l *BaseLocatr) getValidLocator(locators []string) (string, error) {
	for _, locator := range locators {
		if l.plugin.EvaluateJs(fmt.Sprintf("isValidLocator('%s')", locator)) == "true" {
			return locator, nil
		}
	}
	return "", ErrUnableToFindValidLocator
}

func (l *BaseLocatr) getCurrentUrl() string {
	return l.plugin.EvaluateJs("window.location.href")
}

func (l *BaseLocatr) GetLocatorStr(userReq string) (string, error) {
	log.Println("Loading js scripts")
	if err := l.loadJsScripts(); err != nil {
		return "", ErrUnableToLoadJsScripts
	}

	log.Println("Searching for locator in cache")
	hashingKey := hashLocatorKeyWithUrl(l.getCurrentUrl(), userReq)
	locators, err := readLocatorsFromCache(l.cachePath, hashingKey)
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
		return "", ErrUnableToMinifyHtmlDom
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

	if err = writeLocatorsToCache(l.cachePath, hashingKey, locators); err != nil {
		log.Println(err)
	}
	return validLocator, nil
}
