package plugins

import (
	"errors"
	"fmt"

	"github.com/playwright-community/playwright-go"
	"github.com/vertexcover-io/locatr/golang/internal/constants"
	"github.com/vertexcover-io/locatr/golang/internal/utils"
	"github.com/vertexcover-io/locatr/golang/types"
)

// playwrightPlugin encapsulates browser automation functionality using the Playwright framework.
//
// Attributes:
//   - BrowserName: Name of the browser (e.g. "chromium", "firefox", "webkit", etc.)
type playwrightPlugin struct {
	// Playwright page instance
	page *playwright.Page
	// Name of the browser (e.g. "chromium", "firefox", "webkit", etc.)
	BrowserName string
}

// NewPlaywrightPlugin initializes a new plugin instance with the provided Playwright page.
//
// Parameters:
//   - page: Pointer to a configured Playwright page instance
//
// Returns the initialized Playwright plugin.
func NewPlaywrightPlugin(page *playwright.Page) (*playwrightPlugin, error) {
	browserName := (*page).Context().Browser().BrowserType().Name()
	return &playwrightPlugin{page: page, BrowserName: browserName}, nil
}

// evaluateExpression executes a JavaScript expression in the context of the current page.
// If the script is not attached, it will be attached first.
//
// Parameters:
//   - expression: The JavaScript code to execute
//   - args: Optional arguments to pass to the JavaScript expression
//
// Returns the result of the evaluation and any error that occurred during execution.
func (plugin *playwrightPlugin) evaluateExpression(expression string, args ...any) (any, error) {
	isAttached, err := (*plugin.page).Evaluate("() => window.locatrScriptAttached === true")
	if err != nil || isAttached == nil || !isAttached.(bool) {
		if _, err := (*plugin.page).AddScriptTag(
			playwright.PageAddScriptTagOptions{Content: &constants.JS_CONTENT},
		); err != nil {
			return nil, fmt.Errorf("could not add JS content: %v", err)
		}
	}

	result, err := (*plugin.page).Evaluate(expression, args...)
	if err != nil {
		return nil, fmt.Errorf("error evaluating `%v` expression: %v", expression, err)
	}
	return result, nil
}

// GetCurrentContext returns the current page URL.
// This can be used to track the current navigation state of the browser.
func (plugin *playwrightPlugin) GetCurrentContext() (*string, error) {
	url := (*plugin.page).URL()
	return &url, nil
}

// GetMinifiedDOM returns a simplified representation of the current page's DOM structure.
// The DOM is processed to include only relevant information and includes:
//   - A tree structure of elements with their properties
//   - A mapping of elements to their CSS selectors
//
// Returns the processed DOM structure and any error that occurred during extraction.
func (plugin *playwrightPlugin) GetMinifiedDOM() (*types.DOM, error) {
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

// ExtractFirstUniqueID extracts the first unique ID from the given fragment.
func (plugin *playwrightPlugin) ExtractFirstUniqueID(fragment string) (string, error) {
	return utils.ExtractFirstUniqueHTMLID(fragment)
}

// IsLocatorValid checks if a given CSS selector matches any elements on the page.
//
// Parameters:
//   - locator: The CSS selector to validate
//
// Returns true if the selector matches at least one element, false otherwise.
func (plugin *playwrightPlugin) IsLocatorValid(locator string) (bool, error) {
	value, err := plugin.evaluateExpression(
		"(locator) => isLocatorValid(locator)", locator,
	)
	if err != nil {
		return false, err
	}

	return utils.ParseLocatorValidationResult(value)
}

// SetViewportSize adjusts the browser viewport to the specified dimensions.
//
// Parameters:
//   - width: Viewport width in pixels
//   - height: Viewport height in pixels
//
// Returns an error if the viewport adjustment fails.
func (plugin *playwrightPlugin) SetViewportSize(width, height int) error {
	return (*plugin.page).SetViewportSize(width, height)
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

// GetElementLocators retrieves the locators for the element at the given point and scroll position.
//
// Parameters:
//   - location: The location of the element to get the locators from
//
// Returns an array of CSS selectors for elements found at the specified point.
func (plugin *playwrightPlugin) GetElementLocators(location *types.Location) ([]string, error) {
	if location == nil {
		return nil, errors.New("location is required")
	}

	result, err := plugin.evaluateExpression(
		"([x, y, scroll_x, scroll_y]) => getLocators(x, y, scroll_x, scroll_y)",
		[]float64{location.Point.X, location.Point.Y, location.ScrollPosition.X, location.ScrollPosition.Y},
	)
	if err != nil {
		return nil, fmt.Errorf("couldn't get locators from given location: %v", err)
	}
	return utils.ParseLocators(result)
}

// GetElementLocation retrieves the location of the element identified by the given locator.
//
// Parameters:
//   - locator: The CSS selector identifying the target element
//
// Returns the location of the element and any error that occurred during retrieval.
func (plugin *playwrightPlugin) GetElementLocation(locator string) (*types.Location, error) {
	result, err := plugin.evaluateExpression(
		"(locator) => getLocation(locator)", locator,
	)
	if err != nil {
		return nil, fmt.Errorf("couldn't get location from given locator: %v", err)
	}

	return utils.ParseLocation(result)
}
