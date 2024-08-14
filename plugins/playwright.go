package plugins

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/playwright-community/playwright-go"
	"github.com/vertexcover-io/locatr/locatr"
)

type playwrightPlugin struct {
	locatr.PluginInteface
	page playwright.Page
}

type playwrightLocator struct {
	page   playwright.Page
	locatr *locatr.BaseLocatr
}

func NewPlaywrightLocatr(page playwright.Page, llmClient locatr.LlmClient) *playwrightLocator {
	pwPlugin := &playwrightPlugin{page: page}

	return &playwrightLocator{
		page:   page,
		locatr: locatr.NewBaseLocatr(pwPlugin, llmClient),
	}
}

func (pl *playwrightLocator) GetLocatr(userReq string) (playwright.Locator, error) {
	pl.page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateDomcontentloaded,
	})

	locatorStr, err := pl.locatr.GetLocatorStr(userReq)
	if err != nil {
		return nil, err
	}
	return pl.page.Locator(locatorStr), nil
}

func (pl *playwrightPlugin) LoadJsScript(scriptPath string) error {
	scriptContent, err := os.ReadFile(scriptPath)
	if err != nil {
		return err
	}
	pl.page.Evaluate(string(scriptContent))
	return nil
}

func (pl *playwrightPlugin) GetMinifiedDom() (*locatr.ElementSpec, error) {
	evaluationResult, err := pl.page.Evaluate("minifyHTML()")
	if err != nil {
		return nil, err
	}

	minifiedDomStr := evaluationResult.(string)
	minifiedDom := &locatr.ElementSpec{}
	if err = json.Unmarshal([]byte(minifiedDomStr), minifiedDom); err != nil {
		return nil, err
	}

	return minifiedDom, nil
}

func (pl *playwrightPlugin) ExtractIdLocatorMap() (locatr.IdToLocatorMap, error) {
	evaluationResult, err := pl.page.Evaluate("getElementIdLocatorMap()")
	if err != nil {
		return nil, err
	}

	idLocatorMapStr := evaluationResult.(string)
	idLocatorMap := &locatr.IdToLocatorMap{}
	if err = json.Unmarshal([]byte(idLocatorMapStr), idLocatorMap); err != nil {
		return nil, err
	}

	return *idLocatorMap, nil
}

func (pl *playwrightPlugin) GetValidLocator(locators []string) (string, error) {
	for _, locator := range locators {
		count, err := pl.page.Locator(locator).Count()
		if err != nil {
			continue
		}
		if count == 1 {
			return locator, nil
		}
	}
	return "", errors.New("no valid locator found")
}
