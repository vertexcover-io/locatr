package types

// WaitForExpressionOpts defines options for waiting for a JavaScript expression to evaluate to true.
type WaitForExpressionOpts struct {
	Timeout  *float64 // Maximum time to wait in milliseconds
	Interval *float64 // Interval between checks in milliseconds
}

// ScrollPosition is an alias for Point, representing a scroll position in a 2D space.
type ScrollPosition = Point

// PluginInterface defines the interface for a plugin that interacts with a web page.
type PluginInterface interface {

	// EvaluateExpression evaluates a given JavaScript expression with optional arguments.
	EvaluateExpression(expression string, args ...any) (any, error)

	// WaitForExpression waits for a JavaScript expression to evaluate to true.
	WaitForExpression(expression string, args []any, options *WaitForExpressionOpts) error

	// WaitForLoadEvent waits for the page load event to complete.
	WaitForLoadEvent(options *WaitForExpressionOpts) error

	// GetCurrentContext retrieves the current context of the plugin.
	GetCurrentContext() string

	// SetViewportSize sets the size of the viewport.
	SetViewportSize(width, height int) error

	// GetMinifiedDOM retrieves the minified DOM of the current context.
	GetMinifiedDOM() (*DOM, error)

	// ScrollToLocator scrolls to the element identified by the given locator.
	ScrollToLocator(locator string) (*ScrollPosition, error)

	// GetLocatorsFromPoint retrieves locators from a given point on the page.
	// If position is nil, the current viewport position will be used.
	GetLocatorsFromPoint(point *Point, position *ScrollPosition) ([]string, error)

	// TakeScreenshot captures a screenshot of the current viewport.
	TakeScreenshot() ([]byte, error)

	// IsLocatorValid verifies if the given locator is valid.
	IsLocatorValid(locator string) (bool, error)
}
