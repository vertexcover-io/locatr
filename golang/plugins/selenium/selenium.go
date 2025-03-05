package seleniumPlugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/vertexcover-io/locatr/golang/constants"
	"github.com/vertexcover-io/locatr/golang/types"
	"github.com/vertexcover-io/selenium"
)

// seleniumPlugin encapsulates browser automation functionality using the Selenium WebDriver.
// It maintains a reference to a WebDriver instance for performing browser operations.
type seleniumPlugin struct {
	driver selenium.WebDriver
}

// New initializes a new seleniumPlugin instance with the provided Selenium WebDriver.
// Parameters:
//   - driver: Pointer to a configured Selenium WebDriver instance
//
// Returns the initialized plugin and any error that occurred during setup.
func New(driver *selenium.WebDriver) (*seleniumPlugin, error) {
	plugin := &seleniumPlugin{driver: *driver}
	return plugin, nil
}

// EvaluateExpression executes a JavaScript expression in the context of the current page.
// First checks if required JavaScript utilities are loaded, then evaluates the provided expression.
// Parameters:
//   - expression: The JavaScript code to execute
//   - args: Optional arguments to pass to the JavaScript expression
//
// Returns the result of the evaluation and any error that occurred during execution.
func (plugin *seleniumPlugin) EvaluateExpression(expression string, args ...any) (any, error) {
	// Check if script is already attached
	isAttached, err := plugin.driver.ExecuteScript("return window.locatrScriptAttached === true", []any{})
	if err != nil || isAttached == nil || !isAttached.(bool) {
		// Inject the script if not already present
		_, err := plugin.driver.ExecuteScript(constants.JS_CONTENT, []any{})
		if err != nil {
			return nil, fmt.Errorf("could not add JS content: %v", err)
		}
	}

	result, err := plugin.driver.ExecuteScript(fmt.Sprintf("return %s", expression), args)
	if err != nil {
		return nil, fmt.Errorf("error evaluating `%v` expression: %v", expression, err)
	}
	return result, nil
}

// WaitForExpression polls a JavaScript expression until it evaluates to true or times out.
// Parameters:
//   - expression: The JavaScript expression to evaluate
//   - args: Arguments to pass to the expression
//   - options: Configuration options for timeout and polling interval
//
// Returns an error if the condition is not met within the timeout period or if evaluation fails.
func (plugin *seleniumPlugin) WaitForExpression(expression string, args []any, options *types.WaitForExpressionOpts) error {
	timeout := 30000 // Default timeout in milliseconds
	interval := 100  // Default interval in milliseconds

	if options != nil {
		if options.Timeout != nil {
			timeout = int(*options.Timeout)
		}
		if options.Interval != nil {
			interval = int(*options.Interval)
		}
	}

	startTime := time.Now()
	deadline := startTime.Add(time.Duration(timeout) * time.Millisecond)

	for time.Now().Before(deadline) {
		// Evaluate the expression directly using EvaluateExpression
		result, err := plugin.EvaluateExpression(expression, args...)
		if err != nil {
			return fmt.Errorf("error evaluating condition expression '%s': %v", expression, err)
		}

		// Check if the result is truthy
		// Handle different types that could be returned from JavaScript
		isTruthy := false

		switch v := result.(type) {
		case bool:
			isTruthy = v
		case string:
			isTruthy = v != ""
		case float64:
			isTruthy = v != 0
		case int:
			isTruthy = v != 0
		case nil:
			isTruthy = false
		default:
			// For objects, arrays, etc., their existence is truthy
			isTruthy = true
		}

		if isTruthy {
			return nil // Condition met, return success
		}

		// Wait for the interval before trying again
		time.Sleep(time.Duration(interval) * time.Millisecond)
	}

	return fmt.Errorf("condition not met within specified timeout of %dms", timeout)
}

// WaitForLoadEvent waits for the page's document.readyState to become 'complete'.
// This ensures that the page and all its resources have finished loading.
// Parameters:
//   - options: Configuration options for timeout and polling interval
//
// Returns an error if the page doesn't load within the specified timeout.
func (plugin *seleniumPlugin) WaitForLoadEvent(options *types.WaitForExpressionOpts) error {
	return plugin.WaitForExpression("document.readyState === 'complete'", nil, options)
}

// GetCurrentContext returns the current page URL.
// Returns an empty string if the URL cannot be retrieved.
func (plugin *seleniumPlugin) GetCurrentContext() string {
	value, err := plugin.driver.CurrentURL()
	if err != nil {
		return ""
	}
	return value
}

// SetViewportSize adjusts the browser window size to achieve the desired viewport dimensions.
// Accounts for browser chrome (toolbars, scrollbars) when calculating the final window size.
// Parameters:
//   - width: Desired viewport width in pixels
//   - height: Desired viewport height in pixels
//
// Returns an error if the window resize operation fails.
func (plugin *seleniumPlugin) SetViewportSize(width, height int) error {
	handle, err := plugin.driver.CurrentWindowHandle()
	if err != nil {
		return err
	}
	sizeInterface, err := plugin.EvaluateExpression(
		"{width: window.outerWidth - window.innerWidth + arguments[0], height: window.outerHeight - window.innerHeight + arguments[1]};",
		width, height,
	)
	if err != nil {
		return err
	}
	size := sizeInterface.(map[string]any)
	reserr := plugin.driver.ResizeWindow(handle, int(size["width"].(float64)), int(size["height"].(float64)))
	if reserr != nil {
		return reserr
	}
	// TODO: Remove this
	usize, err := plugin.EvaluateExpression("[window.innerWidth, window.innerHeight];")
	if err != nil {
		return err
	}
	log.Println("Updated size:", usize)
	/////////////////////////////////////
	return nil
}

// GetMinifiedDOM returns a simplified representation of the current page's DOM structure.
// The DOM is processed to include only relevant information and includes:
//   - A tree structure of elements with their properties
//   - A mapping of elements to their CSS selectors
//
// Returns the processed DOM structure and any error that occurred during extraction.
func (plugin *seleniumPlugin) GetMinifiedDOM() (*types.DOM, error) {
	result, err := plugin.EvaluateExpression("minifyHTML()")
	if err != nil {
		return nil, err
	}

	rootElement := &types.ElementSpec{}
	resultStr, ok := result.(string)
	if !ok {
		return nil, fmt.Errorf("unexpected type for minified DOM result: %T", result)
	}

	if err := json.Unmarshal([]byte(resultStr), rootElement); err != nil {
		return nil, fmt.Errorf("failed to unmarshal minified root element json: %v", err)
	}

	result, err = plugin.EvaluateExpression("createLocatorMap()")
	if err != nil {
		return nil, err
	}

	locatorMapStr, ok := result.(string)
	if !ok {
		return nil, fmt.Errorf("unexpected type for locator map result: %T", result)
	}

	locatorMap := map[string][]string{}
	if err := json.Unmarshal([]byte(locatorMapStr), &locatorMap); err != nil {
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
// Uses JavaScript scrolling implementation for consistent behavior.
// Parameters:
//   - locator: CSS selector identifying the target element
//
// Returns the final scroll position and any error that occurred during scrolling.
func (plugin *seleniumPlugin) ScrollToLocator(locator string) (*types.ScrollPosition, error) {
	_, err := plugin.EvaluateExpression("scrollToLocator(arguments[0])", locator)
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
// Optionally scrolls to a specific position before checking for elements.
// Parameters:
//   - point: The x,y coordinates to check for elements
//   - position: Optional scroll position to apply before checking coordinates
//
// Returns an array of CSS selectors for elements found at the specified point.
func (plugin *seleniumPlugin) GetLocatorsFromPoint(point *types.Point, position *types.ScrollPosition) ([]string, error) {
	if point == nil {
		return nil, errors.New("point is required")
	}
	if position != nil {
		_, err := plugin.EvaluateExpression(
			"scrollToPosition(...arguments)", position.X, position.Y,
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
		"getLocatorsFromPoint(...arguments)", point.X, point.Y,
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

// TakeScreenshot captures the current viewport as a PNG image using Selenium's Screenshot API.
// Returns the screenshot as a byte array and any error that occurred during capture.
func (plugin *seleniumPlugin) TakeScreenshot() ([]byte, error) {
	bytes, err := plugin.driver.Screenshot()
	if err != nil {
		return nil, fmt.Errorf("could not take screenshot: %v", err)
	}
	return bytes, nil
}

// IsLocatorValid checks if a given CSS selector matches any elements on the page.
// Parameters:
//   - locator: The CSS selector to validate
//
// Returns:
//   - bool: true if the selector matches at least one element, false otherwise
//   - error: any error that occurred during validation
func (plugin *seleniumPlugin) IsLocatorValid(locator string) (bool, error) {
	value, err := plugin.EvaluateExpression("isLocatorValid(arguments[0])", locator)
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
