package playwrightPlugin

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/playwright-community/playwright-go"
	"github.com/vertexcover-io/locatr/golang/constants"
	"github.com/vertexcover-io/locatr/golang/types"
)

// playwrightPlugin encapsulates browser automation functionality using the Playwright framework.
// It maintains a reference to a Playwright page instance for performing browser operations.
type playwrightPlugin struct {
	page *playwright.Page
}

// ensureScript ensures the required JavaScript content is loaded in the page.
// Checks if the script is already attached before injecting to avoid duplicates.
// Returns error if script injection fails.
func (plugin *playwrightPlugin) ensureScript() error {
	isAttached, err := (*plugin.page).Evaluate("() => window.locatrScriptAttached === true")
	if err != nil || isAttached == nil || !isAttached.(bool) {
		_, err := (*plugin.page).AddScriptTag(playwright.PageAddScriptTagOptions{
			Content: &constants.JS_CONTENT,
		})
		if err != nil {
			return fmt.Errorf("could not add JS content: %v", err)
		}
	}
	return nil
}

// New initializes a new playwrightPlugin instance with the provided Playwright page.
func New(page *playwright.Page) (*playwrightPlugin, error) {
	return &playwrightPlugin{page: page}, nil
}

// EvaluateExpression executes a JavaScript expression in the context of the current page.
// Parameters:
//   - expression: The JavaScript code to execute
//   - args: Optional arguments to pass to the JavaScript expression
//
// Returns the result of the evaluation and any error that occurred during execution.
func (plugin *playwrightPlugin) EvaluateExpression(expression string, args ...any) (any, error) {
	if err := plugin.ensureScript(); err != nil {
		return nil, err
	}

	result, err := (*plugin.page).Evaluate(expression, args...)
	if err != nil {
		return nil, fmt.Errorf("error evaluating `%v` expression: %v", expression, err)
	}
	return result, nil
}

// WaitForExpression waits for a JavaScript expression to evaluate to true.
// Parameters:
//   - expression: The JavaScript expression to evaluate
//   - args: Arguments to pass to the expression
//   - options: Configuration options for timeout and polling interval
//
// Returns an error if the wait operation times out or fails.
func (plugin *playwrightPlugin) WaitForExpression(expression string, args []any, options *types.WaitForExpressionOpts) error {
	if err := plugin.ensureScript(); err != nil {
		return err
	}

	waitOpts := playwright.PageWaitForFunctionOptions{}
	if options != nil {
		if options.Timeout != nil {
			waitOpts.Timeout = options.Timeout
		}
		if options.Interval != nil {
			waitOpts.Polling = options.Interval
		}
	}
	_, err := (*plugin.page).WaitForFunction(expression, args, waitOpts)
	return err
}

// WaitForLoadEvent waits for the page's load event to fire.
// This ensures that all resources (images, stylesheets, etc.) have been loaded.
// Parameters:
//   - options: Configuration options for timeout
//
// Returns an error if the wait operation times out or fails.
func (plugin *playwrightPlugin) WaitForLoadEvent(options *types.WaitForExpressionOpts) error {
	stateOpts := playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateLoad,
	}
	if options != nil {
		stateOpts.Timeout = options.Timeout
	}
	return (*plugin.page).WaitForLoadState(stateOpts)
}

// GetCurrentContext returns the current page URL.
// This can be used to track the current navigation state of the browser.
func (plugin *playwrightPlugin) GetCurrentContext() string {
	return (*plugin.page).URL()
}

// SetViewportSize adjusts the browser viewport to the specified dimensions.
// Parameters:
//   - width: Viewport width in pixels
//   - height: Viewport height in pixels
//
// Returns an error if the viewport adjustment fails.
func (plugin *playwrightPlugin) SetViewportSize(width, height int) error {
	return (*plugin.page).SetViewportSize(width, height)
}

// GetMinifiedDOM returns a simplified representation of the current page's DOM structure.
// The DOM is processed to include only relevant information and includes:
//   - A tree structure of elements with their properties
//   - A mapping of elements to their CSS selectors
//
// Returns the processed DOM structure and any error that occurred during extraction.
func (plugin *playwrightPlugin) GetMinifiedDOM() (*types.DOM, error) {
	result, err := plugin.EvaluateExpression("minifyHTML()")
	if err != nil {
		return nil, err
	}

	rootElement := &types.ElementSpec{}
	if err := json.Unmarshal([]byte(result.(string)), rootElement); err != nil {
		return nil, fmt.Errorf("failed to unmarshal minified root element json: %v", err)
	}

	result, _ = plugin.EvaluateExpression("createLocatorMap()")
	locatorMap := map[string][]string{}
	if err := json.Unmarshal([]byte(result.(string)), &locatorMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal locator map json: %v", err)
	}

	return &types.DOM{
		RootElement: rootElement,
		Metadata: &types.DOMMetadata{
			LocatorType: types.CssSelectorType, LocatorMap: locatorMap,
		},
	}, nil
}

// ScrollToLocator scrolls the page to bring the specified element into view.
// Parameters:
//   - locator: CSS selector identifying the target element
//
// Returns the final scroll position and any error that occurred during scrolling.
func (plugin *playwrightPlugin) ScrollToLocator(locator string) (*types.ScrollPosition, error) {
	_, err := plugin.EvaluateExpression(
		"(locator) => scrollToLocator(locator)", locator,
	)
	if err != nil {
		return nil, fmt.Errorf("error occured while scrolling to locator: %v", err)
	}

	_, err = plugin.EvaluateExpression("waitForScrollCompletion()")
	if err != nil {
		return nil, fmt.Errorf("error occured while waiting for scroll completion: %v", err)
	}
	scrollPositionInterface, err := plugin.EvaluateExpression("getScrollPosition()")
	if err != nil {
		return nil, fmt.Errorf("could not get scroll position: %v", err)
	}
	var getFloatValue = func(v any) float64 {
		switch t := v.(type) {
		case float64:
			return t
		default:
			return float64(v.(int))
		}
	}
	scrollPositionMap := scrollPositionInterface.(map[string]any)
	scrollPosition := &types.ScrollPosition{
		X: getFloatValue(scrollPositionMap["x"]),
		Y: getFloatValue(scrollPositionMap["y"]),
	}
	return scrollPosition, nil
}

// GetLocatorsFromPoint retrieves CSS selectors for elements at the specified coordinates.
// Parameters:
//   - point: The x,y coordinates to check for elements
//   - position: Optional scroll position to apply before checking coordinates
//
// Returns an array of CSS selectors for elements found at the specified point.
func (plugin *playwrightPlugin) GetLocatorsFromPoint(point *types.Point, position *types.ScrollPosition) ([]string, error) {
	if point == nil {
		return nil, errors.New("point is required")
	}
	if position != nil {
		_, err := plugin.EvaluateExpression(
			"([x, y]) => scrollToPosition(x, y)", []float64{position.X, position.Y},
		)
		if err != nil {
			return nil, fmt.Errorf("error occured while scrolling to position: %v", err)
		}
		_, err = plugin.EvaluateExpression("waitForScrollCompletion()")
		if err != nil {
			return nil, fmt.Errorf("error occured while waiting for scroll completion: %v", err)
		}
	}
	result, err := plugin.EvaluateExpression(
		"([x, y]) => getLocatorsFromPoint(x, y)", []float64{point.X, point.Y},
	)
	if err != nil {
		return nil, fmt.Errorf("couldn't get locators from given point: %v", err)
	}
	interfaceSlice := result.([]any)
	stringSlice := make([]string, len(interfaceSlice))
	for i, v := range interfaceSlice {
		stringSlice[i] = v.(string)
	}
	return stringSlice, nil
}

// TakeScreenshot captures the current viewport as a PNG image.
// Returns the screenshot as a byte array and any error that occurred during capture.
func (plugin *playwrightPlugin) TakeScreenshot() ([]byte, error) {
	bytes, err := (*plugin.page).Screenshot()
	if err != nil {
		return nil, fmt.Errorf("could not take screenshot: %v", err)
	}
	return bytes, nil
}

// IsLocatorValid checks if a given CSS selector matches any elements on the page.
// Parameters:
//   - locator: The CSS selector to validate
//
// Returns true if the selector matches at least one element, false otherwise.
func (plugin *playwrightPlugin) IsLocatorValid(locator string) (bool, error) {
	value, err := plugin.EvaluateExpression(
		"([locator]) => isLocatorValid(locator)", []string{locator},
	)
	if err != nil {
		return false, err
	}

	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		var isValid bool
		if err := json.Unmarshal([]byte(v), &isValid); err != nil {
			return false, fmt.Errorf("failed to parse locator validation result: %v", err)
		}
		return isValid, nil
	default:
		return false, fmt.Errorf("unexpected type for locator validation result: %T", value)
	}
}
