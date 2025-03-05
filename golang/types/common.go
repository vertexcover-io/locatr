package types

type locatorType = string

// Constants for different types of locators.
const (
	CssSelectorType locatorType = "css-selector"
	XPathType       locatorType = "xpath"
)

// DOMMetadata holds metadata information about a DOM, including locator types and mappings.
type DOMMetadata struct {
	LocatorType locatorType         // Type of the locators in the map
	LocatorMap  map[string][]string // Mapping of element IDs to their locators
}

// DOM represents the structure of a Document Object Model (DOM).
type DOM struct {
	RootElement *ElementSpec // The root element of the DOM
	Metadata    *DOMMetadata // Metadata associated with the DOM
}

// Point represents a coordinate point in a 2D space.
type Point struct {
	X float64 `json:"x"` // X-coordinate
	Y float64 `json:"y"` // Y-coordinate
}

// LLMProvider is a string alias representing a language model provider.
type LLMProvider = string

// CacheEntry represents a cache entry for storing locator information.
type CacheEntry struct {
	UserRequest string      `json:"locatr_name"`   // User's request or query name
	Locators    []string    `json:"locatrs"`       // List of locators associated with the request
	LocatorType locatorType `json:"selector_type"` // Type of locator used
}
