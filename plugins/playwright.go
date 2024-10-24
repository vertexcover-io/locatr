package plugins

import (
	"fmt"

	"github.com/playwright-community/playwright-go"
	"github.com/vertexcover-io/locatr/locatr"
)

type playwrightPlugin struct {
	page playwright.Page
	locatr.PluginInterface
}

type playwrightLocator struct {
	page   playwright.Page
	locatr *locatr.BaseLocatr
}

func NewPlaywrightLocatr(page playwright.Page, llmClient locatr.LlmClient, options locatr.BaseLocatrOptions) *playwrightLocator {
	pwPlugin := &playwrightPlugin{page: page}

	return &playwrightLocator{
		page:   page,
		locatr: locatr.NewBaseLocatr(pwPlugin, llmClient, options),
	}
}

func (pl *playwrightPlugin) EvaluateJsFunction(function string) string {
	result, err := pl.page.Evaluate(function)
	if err != nil {
		return ""
	}
	if result == nil {
		return ""
	}

	if str, ok := result.(string); ok {
		return str
	}
	if num, ok := result.(float64); ok {
		return fmt.Sprint(num)
	}
	if boolval, ok := result.(bool); ok {
		return fmt.Sprint(boolval)
	}
	return ""
}

func (pl *playwrightPlugin) EvaluateJsScript(scriptContent string) error {
	if _, err := pl.page.Evaluate(string(scriptContent)); err != nil {
		return err
	}
	return nil
}

func (pl *playwrightLocator) GetLocatr(userReq string) (playwright.Locator, error) {
	if err := pl.page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{State: playwright.LoadStateDomcontentloaded}); err != nil {
		return nil, err
	}

	locatorStr, err := pl.locatr.GetLocatorStr(userReq)
	if err != nil {
		return nil, err
	}
	return pl.page.Locator(locatorStr), nil
}
