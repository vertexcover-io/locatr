package plugins

import (
	"errors"
	"fmt"
	"log"

	"github.com/vertexcover-io/locatr/golang/internal/constants"
	"github.com/vertexcover-io/locatr/golang/internal/utils"
	"github.com/vertexcover-io/locatr/golang/types"
	"github.com/vertexcover-io/selenium"
)

// seleniumPlugin encapsulates browser automation functionality using the Selenium WebDriver.
type seleniumPlugin struct {
	// Selenium WebDriver instance
	driver *selenium.WebDriver
}

// NewSeleniumPlugin initializes a new seleniumPlugin instance with the provided Selenium WebDriver.
// Parameters:
//   - driver: Pointer to a configured Selenium WebDriver instance
//
// Returns the initialized plugin.
func NewSeleniumPlugin(driver *selenium.WebDriver) *seleniumPlugin {
	return &seleniumPlugin{driver: driver}
}

// evaluateExpression executes a JavaScript expression in the current context.
// If the script is not attached, it will be attached first.
// Parameters:
//   - expression: The JavaScript code to execute
//   - args: Optional arguments to pass to the JavaScript expression
//
// Returns the result of the evaluation and any error that occurred during execution.
func (plugin *seleniumPlugin) evaluateExpression(expression string, args ...any) (any, error) {
	// Check if script is already attached
	isAttached, err := (*plugin.driver).ExecuteScript("return window.locatrScriptAttached === true", []any{})
	if err != nil || isAttached == nil || !isAttached.(bool) {
		_, err := (*plugin.driver).ExecuteScript(constants.JS_CONTENT, []any{})
		if err != nil {
			return nil, fmt.Errorf("could not add JS content: %v", err)
		}
	}

	result, err := (*plugin.driver).ExecuteScript(fmt.Sprintf("return %s", expression), args)
	if err != nil {
		return nil, fmt.Errorf("error evaluating `%v` expression: %v", expression, err)
	}
	return result, nil
}

// GetCurrentContext returns the current page URL.
// Returns an empty string if the URL cannot be retrieved.
func (plugin *seleniumPlugin) GetCurrentContext() (*string, error) {
	value, err := (*plugin.driver).CurrentURL()
	if err != nil {
		return nil, err
	}
	return &value, nil
}

// GetMinifiedDOM returns a simplified representation of the current page's DOM structure.
// The DOM is processed to include only relevant information and includes:
//   - A tree structure of elements with their properties
//   - A mapping of elements to their CSS selectors
//
// Returns the processed DOM structure and any error that occurred during extraction.
func (plugin *seleniumPlugin) GetMinifiedDOM() (*types.DOM, error) {
	result, err := plugin.evaluateExpression("minifyHTML()")
	if err != nil {
		return nil, fmt.Errorf("couldn't get minified DOM: %v", err)
	}

	rootElement, err := utils.ParseElementSpec(result)
	if err != nil {
		return nil, err
	}

	result, err = plugin.evaluateExpression("createLocatorMap()")
	if err != nil {
		return nil, fmt.Errorf("couldn't get locator map: %v", err)
	}

	locatorMap, err := utils.ParseLocatorMap(result)
	if err != nil {
		return nil, err
	}

	dom := &types.DOM{
		RootElement: rootElement,
		Metadata: &types.DOMMetadata{
			LocatorType: types.CssSelectorType, LocatorMap: locatorMap,
		},
	}
	return dom, nil
}

// IsLocatorValid checks if a given CSS selector matches any elements on the page.
// Parameters:
//   - locator: The CSS selector to validate
//
// Returns:
//   - bool: true if the selector matches at least one element, false otherwise
//   - error: any error that occurred during validation
func (plugin *seleniumPlugin) IsLocatorValid(locator string) (bool, error) {
	value, err := plugin.evaluateExpression("isLocatorValid(arguments[0])", locator)
	if err != nil {
		return false, err
	}

	return utils.ParseLocatorValidationResult(value)
}

// SetViewportSize adjusts the browser window size to achieve the desired viewport dimensions.
// Accounts for browser chrome (toolbars, scrollbars) when calculating the final window size.
// Parameters:
//   - width: Desired viewport width in pixels
//   - height: Desired viewport height in pixels
//
// Returns an error if the window resize operation fails.
func (plugin *seleniumPlugin) SetViewportSize(width, height int) error {
	handle, err := (*plugin.driver).CurrentWindowHandle()
	if err != nil {
		return err
	}
	sizeInterface, err := plugin.evaluateExpression(
		"{width: window.outerWidth - window.innerWidth + arguments[0], height: window.outerHeight - window.innerHeight + arguments[1]};",
		width, height,
	)
	if err != nil {
		return err
	}
	size := sizeInterface.(map[string]any)
	reserr := (*plugin.driver).ResizeWindow(handle, int(size["width"].(float64)), int(size["height"].(float64)))
	if reserr != nil {
		return reserr
	}
	// TODO: Remove this
	usize, err := plugin.evaluateExpression("[window.innerWidth, window.innerHeight];")
	if err != nil {
		return err
	}
	log.Println("Updated size:", usize)
	/////////////////////////////////////
	return nil
}

// TakeScreenshot captures the current viewport as a PNG image using Selenium's Screenshot API.
// Returns the screenshot as a byte array and any error that occurred during capture.
func (plugin *seleniumPlugin) TakeScreenshot() ([]byte, error) {
	bytes, err := (*plugin.driver).Screenshot()
	if err != nil {
		return nil, fmt.Errorf("could not take screenshot: %v", err)
	}
	return bytes, nil
}

// GetElementLocators retrieves the locators for the element at the given point and scroll position.
// Parameters:
//   - location: The location of the element to get the locators from
//
// Returns an array of CSS selectors for elements found at the specified point.
func (plugin *seleniumPlugin) GetElementLocators(location *types.Location) ([]string, error) {
	if location == nil {
		return nil, errors.New("location is required")
	}

	result, err := plugin.evaluateExpression(
		"getLocators(arguments...)",
		location.Point.X, location.Point.Y, location.ScrollPosition.X, location.ScrollPosition.Y,
	)
	if err != nil {
		return nil, fmt.Errorf("couldn't get locators from given location: %v", err)
	}
	return utils.ParseLocators(result)
}

// GetElementLocation retrieves the location of the element identified by the given locator.
// Parameters:
//   - locator: The CSS selector identifying the target element
//
// Returns the location of the element and any error that occurred during retrieval.
func (plugin *seleniumPlugin) GetElementLocation(locator string) (*types.Location, error) {
	result, err := plugin.evaluateExpression(
		"getLocation(arguments[0])", locator,
	)
	if err != nil {
		return nil, fmt.Errorf("couldn't get location from given locator: %v", err)
	}

	return utils.ParseLocation(result)
}
