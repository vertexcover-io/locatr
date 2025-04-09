package types

import "context"

// Location represents a location on a web page, consisting of a point and a scroll position.
type Location struct {
	// Point is the position of the element on the viewport.
	Point Point `json:"point"`
	// ScrollPosition is the position of the scroll on the page.
	ScrollPosition Point `json:"scroll_position"`
}

// PluginInterface defines the interface for a plugin that interacts with a web page.
type PluginInterface interface {

	// GetCurrentContext retrieves the current context of the plugin.
	GetCurrentContext(ctx context.Context) (*string, error)

	// GetMinifiedDOM retrieves the minified DOM and associated metadata of the current context.
	GetMinifiedDOM(ctx context.Context) (*DOM, error)

	// ExtractFirstUniqueID extracts the first unique ID from the given fragment.
	ExtractFirstUniqueID(ctx context.Context, fragment string) (string, error)

	// IsLocatorValid verifies if the given locator is valid.
	IsLocatorValid(ctx context.Context, locator string) (bool, error)

	// SetViewportSize sets the size of the viewport.
	SetViewportSize(ctx context.Context, width, height int) error

	// TakeScreenshot captures a screenshot of the current viewport.
	TakeScreenshot(ctx context.Context) ([]byte, error)

	// GetElementLocators retrieves locators from a given point and scroll position on the page.
	// If scroll position is nil, the current viewport position will be used.
	GetElementLocators(ctx context.Context, location *Location) ([]string, error)

	// GetElementLocation retrieves the point and scroll position of the element identified by the given locator.
	GetElementLocation(ctx context.Context, locator string) (*Location, error)
}
