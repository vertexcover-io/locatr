package locatr

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
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

type BaseLocatr struct {
	plugin    PluginInterface
	llmClient LlmClient
}

func NewBaseLocatr(plugin PluginInterface, llmClient LlmClient) *BaseLocatr {
	return &BaseLocatr{
		plugin:    plugin,
		llmClient: llmClient,
	}
}

func (al *BaseLocatr) locateElementId(htmlDOM string, userReq string) (string, error) {
	systemPrompt, err := os.ReadFile("meta/locate_element.prompt")
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
	if err := l.plugin.LoadJsScript("meta/htmlMinifier.js"); err != nil {
		return "", ErrUnableToLoadJsScripts
	}

	minifiedDOM, err := l.plugin.GetMinifiedDom()
	if err != nil {
		log.Println(err)
		return "", ErrUnableToMinifyHtmlDom
	}

	locatorsMap, err := l.plugin.ExtractIdLocatorMap()
	if err != nil {
		log.Println(err)
		return "", ErrUnableToExtractIdLocatorMap
	}

	elementID, err := l.locateElementId(minifiedDOM.ContentStr(), userReq)
	if err != nil {
		log.Println(err)
		return "", ErrUnableToLocateElementId
	}

	if locators, ok := locatorsMap[elementID]; ok {
		validLocator, err := l.plugin.GetValidLocator(locators)
		if err != nil {
			log.Println(err)
			return "", ErrUnableToFindValidLocator
		}
		return validLocator, nil
	}
	return "", ErrInvalidElementIdGenerated
}
