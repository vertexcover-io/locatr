package types

// WaitForExpressionOpts defines options for waiting for a JavaScript expression to evaluate to true.
type WaitForExpressionOpts struct {
	Timeout  *float64 // Maximum time to wait in milliseconds
	Interval *float64 // Interval between checks in milliseconds
}

// Location represents a location on a web page, consisting of a point and a scroll position.
type Location struct {
	// Point is the position of the element on the viewport.
	Point Point `json:"point"`

	// ScrollPosition is the position of the scroll on the page.
	ScrollPosition Point `json:"scroll_position"`
}

// PluginInterface defines the interface for a plugin that interacts with a web page.
type PluginInterface interface {

	// WaitForLoadEvent waits for the page load event to complete.
	WaitForLoadEvent(timeout *float64) error

	// GetCurrentContext retrieves the current context of the plugin.
	GetCurrentContext() string

	// SetViewportSize sets the size of the viewport.
	SetViewportSize(width, height int) error

	// GetMinifiedDOM retrieves the minified DOM of the current context.
	GetMinifiedDOM() (*DOM, error)

	// GetLocators retrieves locators from a given point and scroll position on the page.
	// If scroll position is nil, the current viewport position will be used.
	GetLocators(location *Location) ([]string, error)

	// GetLocation retrieves the point and scroll position of the element identified by the given locator.
	GetLocation(locator string) (*Location, error)

	// TakeScreenshot captures a screenshot of the current viewport.
	TakeScreenshot() ([]byte, error)

	// IsLocatorValid verifies if the given locator is valid.
	IsLocatorValid(locator string) (bool, error)
}
