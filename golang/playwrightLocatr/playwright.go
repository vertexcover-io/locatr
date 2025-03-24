package playwrightLocatr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/playwright-community/playwright-go"
	locatr "github.com/vertexcover-io/locatr/golang"
	"github.com/vertexcover-io/locatr/golang/elementSpec"
	"github.com/vertexcover-io/locatr/golang/tracing"
	"go.opentelemetry.io/otel/trace"
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
func NewPlaywrightLocatr(ctx context.Context, page playwright.Page, options locatr.BaseLocatrOptions) *PlaywrightLocator {
	span := trace.SpanFromContext(ctx)

	span.AddEvent("Creating new playwright plugin")

	pwPlugin := &playwrightPlugin{page: page}

	return &PlaywrightLocator{
		page:   page,
		locatr: locatr.NewBaseLocatr(pwPlugin, options),
	}
}

func (pl *playwrightPlugin) GetMinifiedDomAndLocatorMap(ctx context.Context) (
	*elementSpec.ElementSpec,
	*elementSpec.IdToLocatorMap,
	locatr.SelectorType,
	error,
) {
	ctx, span := tracing.StartSpan(ctx, "GetMinifiedDomAndLocatorMap")
	defer span.End()

	span.AddEvent("injecting HTML minifier script")
	if err := pl.evaluateJsScript(ctx, locatr.HTML_MINIFIER_JS_CONTENT); err != nil {
		return nil, nil, "", fmt.Errorf("%v : %v", ErrUnableToLoadJsScriptsThroughPlaywright, err)
	}

	span.AddEvent("evaluating minifyHTML function")
	result, err := pl.evaluateJsFunction(ctx, "minifyHTML()")
	if err != nil {
		return nil, nil, "", err
	}
	elementsSpec := &elementSpec.ElementSpec{}
	if err := json.Unmarshal([]byte(result), elementsSpec); err != nil {
		return nil, nil, "", fmt.Errorf(
			"failed to unmarshal ElementSpec json: %v, expected json, received: %s",
			err,
			result,
		)
	}

	span.AddEvent("evaluating mapElementsToJson function")
	result, _ = pl.evaluateJsFunction(ctx, "mapElementsToJson()")
	idLocatorMap := &elementSpec.IdToLocatorMap{}
	if err := json.Unmarshal([]byte(result), idLocatorMap); err != nil {
		return nil, nil, "", fmt.Errorf(
			"failed to unmarshal IdToLocatorMap json: %v, expected json, received: %s",
			err,
			result,
		)
	}
	return elementsSpec, idLocatorMap, "css selector", nil
}

// evaluateJsFunction runs the given javascript function in the browser and returns the result as a string.
func (pl *playwrightPlugin) evaluateJsFunction(ctx context.Context, function string) (string, error) {
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
func (pl *playwrightPlugin) evaluateJsScript(ctx context.Context, scriptContent string) error {
	if _, err := pl.page.Evaluate(scriptContent); err != nil {
		fmt.Println("here ---")
		return err
	}
	return nil
}

// GetLocatr returns a playwright locator object for the given user request.
func (pl *PlaywrightLocator) GetLocatr(ctx context.Context, userReq string) (*locatr.LocatrOutput, error) {
	ctx, span := tracing.StartSpan(ctx, "GetLocatr")
	defer span.End()

	span.AddEvent("waiting for DOM content to load")
	if err := pl.page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{State: playwright.LoadStateDomcontentloaded}); err != nil {
		return nil, fmt.Errorf("error waiting for load state: %v", err)
	}

	locatorOutput, err := pl.locatr.GetLocatorStr(ctx, userReq)
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
func (pl *playwrightPlugin) GetCurrentContext(ctx context.Context) string {
	ctx, span := tracing.StartSpan(ctx, "GetCurrentContext")
	defer span.End()

	span.AddEvent("fetching current window location")
	if value, err := pl.evaluateJsFunction(ctx, "window.location.href"); err == nil {
		return value
	} else {
		return ""
	}
}

func (pl *playwrightPlugin) IsValidLocator(ctx context.Context, locatrString string) (bool, error) {
	ctx, span := tracing.StartSpan(ctx, "IsValidLocator")
	defer span.End()

	span.AddEvent("injecting HTML minifier script")
	if err := pl.evaluateJsScript(ctx, locatr.HTML_MINIFIER_JS_CONTENT); err != nil {
		return false, fmt.Errorf("%v : %v", ErrUnableToLoadJsScriptsThroughPlaywright, err)
	}

	span.AddEvent("evaluating valid locator function")
	value, err := pl.evaluateJsFunction(ctx, fmt.Sprintf("isValidLocator('%s')", locatrString))
	if value == "true" && err == nil {
		return true, nil
	} else {
		return false, err
	}
}
