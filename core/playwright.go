package core

import (
	"fmt"

	"github.com/playwright-community/playwright-go"
)

type playwrightPlugin struct {
	page playwright.Page
	PluginInterface
}

type playwrightLocator struct {
	page   playwright.Page
	locatr *BaseLocatr
}

// NewPlaywrightLocatr creates a new playwright locator. Use the returned struct methods to get locators.
func NewPlaywrightLocatr(page playwright.Page, llmClient LlmClient, options BaseLocatrOptions) *playwrightLocator {
	pwPlugin := &playwrightPlugin{page: page}

	return &playwrightLocator{
		page:   page,
		locatr: NewBaseLocatr(pwPlugin, llmClient, options),
	}
}

// evaluateJsFunction runs the given javascript function in the browser and returns the result as a string.
func (pl *playwrightPlugin) evaluateJsFunction(function string) (string, error) {
	result, err := pl.page.Evaluate(function)
	if err != nil {
		return "", fmt.Errorf("error evaluating js function: %v", err)
	}
	if result == nil {
		return "", fmt.Errorf("error evaluating js function: result is nil")
	}

	if str, ok := result.(string); ok {
		return str, nil
	}
	if num, ok := result.(float64); ok {
		return fmt.Sprint(num), nil
	}
	if boolval, ok := result.(bool); ok {
		return fmt.Sprint(boolval), nil
	}
	return "", fmt.Errorf("error evaluating js function: result is not string, number or boolean")
}

// evaluateJsScript runs the given javascript script in the browser.
func (pl *playwrightPlugin) evaluateJsScript(scriptContent string) error {
	if _, err := pl.page.Evaluate(string(scriptContent)); err != nil {
		return fmt.Errorf("error evaluating js script: %v", err)
	}
	return nil
}

// GetLocatr returns a pywright locator object for the given user request.
func (pl *playwrightLocator) GetLocatr(userReq string) (playwright.Locator, error) {
	if err := pl.page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{State: playwright.LoadStateDomcontentloaded}); err != nil {
		return nil, fmt.Errorf("error waiting for load state: %v", err)
	}

	locatorStr, err := pl.locatr.getLocatorStr(userReq)
	if err != nil {
		return nil, fmt.Errorf("error getting locator string: %v", err)
	}
	return pl.page.Locator(locatorStr), nil
}
