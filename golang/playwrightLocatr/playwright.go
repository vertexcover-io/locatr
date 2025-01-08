package playwrightLocatr

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/playwright-community/playwright-go"
	locatr "github.com/vertexcover-io/locatr/golang"
	"github.com/vertexcover-io/locatr/golang/elementSpec"
)

type playwrightPlugin struct {
	page playwright.Page
}

type PlaywrightLocator struct {
	page   playwright.Page
	locatr *locatr.BaseLocatr
}

var ErrUnableToLoadJsScriptsThroughPlaywright = errors.New("unable to load js script through playwright")

// NewPlaywrightLocatr creates a new playwright locator. Use the returned struct methods to get locators.
func NewPlaywrightLocatr(page playwright.Page, options locatr.BaseLocatrOptions) *PlaywrightLocator {
	pwPlugin := &playwrightPlugin{page: page}

	return &PlaywrightLocator{
		page:   page,
		locatr: locatr.NewBaseLocatr(pwPlugin, options),
	}
}

func (pl *playwrightPlugin) GetMinifiedDomAndLocatorMap() (
	*elementSpec.ElementSpec,
	*elementSpec.IdToLocatorMap,
	locatr.SelectorType,
	error,
) {
	if err := pl.evaluateJsScript(locatr.HTML_MINIFIER_JS_CONTENT); err != nil {
		return nil, nil, "", fmt.Errorf("%v : %v", ErrUnableToLoadJsScriptsThroughPlaywright, err)
	}
	result, err := pl.evaluateJsFunction("minifyHTML()")
	if err != nil {
		return nil, nil, "", err
	}
	elementsSpec := &elementSpec.ElementSpec{}
	if err := json.Unmarshal([]byte(result), elementsSpec); err != nil {
		return nil, nil, "", fmt.Errorf("failed to unmarshal ElementSpec json: %v", err)
	}

	result, _ = pl.evaluateJsFunction("mapElementsToJson()")
	idLocatorMap := &elementSpec.IdToLocatorMap{}
	if err := json.Unmarshal([]byte(result), idLocatorMap); err != nil {
		return nil, nil, "", fmt.Errorf("failed to unmarshal IdToLocatorMap json: %v", err)
	}
	return elementsSpec, idLocatorMap, "css", nil
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
	if _, err := pl.page.Evaluate(scriptContent); err != nil {
		fmt.Println("here ---")
		return err
	}
	return nil
}

// GetLocatr returns a playwright locator object for the given user request.
func (pl *PlaywrightLocator) GetLocatr(userReq string) (*locatr.LocatrOutput, error) {
	if err := pl.page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{State: playwright.LoadStateDomcontentloaded}); err != nil {
		return nil, fmt.Errorf("error waiting for load state: %v", err)
	}

	locatorOutput, err := pl.locatr.GetLocatorStr(userReq)
	if err != nil {
		return nil, fmt.Errorf("error getting locator string: %v", err)
	}
	return locatorOutput, nil
}

// WriteResultsToFile writes the locatr results to path specified in BaseLocatrOptions.ResultsFilePath.
func (pl *PlaywrightLocator) WriteResultsToFile() {
	pl.locatr.WriteLocatrResultsToFile()
}

// GetLocatrResults returns the locatr results.
func (pl *PlaywrightLocator) GetLocatrResults() []locatr.LocatrResult {
	return pl.locatr.GetLocatrResults()
}
func (pl *playwrightPlugin) GetCurrentContext() string {
	if value, err := pl.evaluateJsFunction("window.location.href"); err == nil {
		return value
	} else {
		return ""
	}
}
func (pl *playwrightPlugin) IsValidLocator(locatrString string) (bool, error) {
	if err := pl.evaluateJsScript(locatr.HTML_MINIFIER_JS_CONTENT); err != nil {
		return false, fmt.Errorf("%v : %v", ErrUnableToLoadJsScriptsThroughPlaywright, err)
	}
	value, err := pl.evaluateJsFunction(fmt.Sprintf("isValidLocator('%s')", locatrString))
	if value == "true" && err == nil {
		return true, nil
	} else {
		return false, err
	}
}
