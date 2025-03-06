package playwrightPlugin

import (
	"errors"
	"fmt"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/vertexcover-io/locatr/golang/constants"
	"github.com/vertexcover-io/locatr/golang/options"
	"github.com/vertexcover-io/locatr/golang/types"
	"github.com/vertexcover-io/locatr/golang/utils"
)

// playwrightPlugin encapsulates browser automation functionality using the Playwright framework.
type playwrightPlugin struct {
	// Playwright page instance
	page *playwright.Page
	// Plugin options
	opts *options.PluginOptions
	// Cached DOM
	cachedDOM *types.DOM
}

// New initializes a new playwrightPlugin instance with the provided Playwright page.
// Parameters:
//   - page: Pointer to a configured Playwright page instance
//   - opts: Options for the plugin

// Returns the initialized plugin.
func New(page *playwright.Page, opts *options.PluginOptions) *playwrightPlugin {
	if opts == nil {
		opts = &options.PluginOptions{}
	}
	if opts.OnContextChangeSleep == 0 {
		opts.OnContextChangeSleep = 3000 // Default to 3 seconds
	}
	return &playwrightPlugin{page: page, opts: opts}
}

// evaluateExpression executes a JavaScript expression in the context of the current page.
// If the script is not attached, it will be attached first.
// Parameters:
//   - expression: The JavaScript code to execute
//   - args: Optional arguments to pass to the JavaScript expression
//
// Returns the result of the evaluation and any error that occurred during execution.
func (plugin *playwrightPlugin) evaluateExpression(expression string, args ...any) (any, error) {
	// Check if script is already attached
	isAttached, err := (*plugin.page).Evaluate("() => window.locatrScriptAttached === true")
	if err != nil || isAttached == nil || !isAttached.(bool) {
		time.Sleep(time.Duration(plugin.opts.OnContextChangeSleep) * time.Millisecond)
		_, err := (*plugin.page).AddScriptTag(playwright.PageAddScriptTagOptions{
			Content: &constants.JS_CONTENT,
		})
		if err != nil {
			return nil, fmt.Errorf("could not add JS content: %v", err)
		}
		plugin.cachedDOM = nil // Remove the cached DOM as it not valid anymore
	}

	result, err := (*plugin.page).Evaluate(expression, args...)
	if err != nil {
		return nil, fmt.Errorf("error evaluating `%v` expression: %v", expression, err)
	}
	return result, nil
}

// WaitForLoadEvent waits for the page's load event to fire.
// This ensures that all resources (images, stylesheets, etc.) have been loaded.
// Parameters:
//   - timeout: Timeout in milliseconds. Default is 30000ms.
//
// Returns an error if the wait operation times out or fails.
func (plugin *playwrightPlugin) WaitForLoadEvent(timeout *float64) error {
	stateOpts := playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateLoad,
	}
	if timeout != nil {
		stateOpts.Timeout = timeout
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
	if plugin.cachedDOM != nil {
		return plugin.cachedDOM, nil
	}

	result, err := plugin.evaluateExpression("minifyHTML()")
	if err != nil {
		return nil, err
	}

	rootElement, err := utils.ParseElementSpec(result)
	if err != nil {
		return nil, err
	}

	result, err = plugin.evaluateExpression("createLocatorMap()")
	if err != nil {
		return nil, err
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
	plugin.cachedDOM = dom
	return dom, nil
}

// GetLocators retrieves the locators for the element at the given point and scroll position.
// Parameters:
//   - location: The location of the element to get the locators from
//
// Returns an array of CSS selectors for elements found at the specified point.
func (plugin *playwrightPlugin) GetLocators(location *types.Location) ([]string, error) {
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

// GetLocation retrieves the location of the element identified by the given locator.
// Parameters:
//   - locator: The CSS selector identifying the target element
//
// Returns the location of the element and any error that occurred during retrieval.
func (plugin *playwrightPlugin) GetLocation(locator string) (*types.Location, error) {
	result, err := plugin.evaluateExpression(
		"(locator) => getLocation(locator)", locator,
	)
	if err != nil {
		return nil, fmt.Errorf("couldn't get location from given locator: %v", err)
	}

	return utils.ParseLocation(result)
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
	value, err := plugin.evaluateExpression(
		"(locator) => isLocatorValid(locator)", locator,
	)
	if err != nil {
		return false, err
	}

	return utils.ParseLocatorValidationResult(value)
}
