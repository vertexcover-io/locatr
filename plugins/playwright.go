package plugins

import (
	"fmt"

	"github.com/playwright-community/playwright-go"
	"github.com/vertexcover-io/locatr/locatr"
)

type playwrightPlugin struct {
	locatr.PluginInterface
	page playwright.Page
}

type playwrightLocator struct {
	page   playwright.Page
	locatr *locatr.BaseLocatr
}

func NewPlaywrightLocatr(page playwright.Page, conf *locatr.LocatrConfig) (*playwrightLocator, error) {
	pwPlugin := &playwrightPlugin{page: page}
	baseLocatr, err := locatr.NewBaseLocatr(pwPlugin, conf)
	if err != nil {
		return nil, err
	}

	return &playwrightLocator{
		page:   page,
		locatr: baseLocatr,
	}, nil
}

func (pl *playwrightPlugin) EvaluateJs(jsStr string) string {
	result, err := pl.page.Evaluate(jsStr)
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
