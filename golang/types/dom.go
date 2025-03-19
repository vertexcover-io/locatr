package types

import "fmt"

type locatorType = string

// Constants for different types of locators.
const (
	CssSelectorType locatorType = "css selector"
	XPathType       locatorType = "xpath"
)

// ElementSpec represents the specification of an element.
type ElementSpec struct {
	Id         string            `json:"id" validate:"required"`       // Unique identifier of the element
	TagName    string            `json:"tag_name" validate:"required"` // Tag name of the element (e.g., div, span)
	Text       string            `json:"text"`                         // Inner text of the element
	Attributes map[string]string `json:"attributes"`                   // Attributes of the element (e.g., class, style)
	Children   []ElementSpec     `json:"children"`                     // Child elements of this element
}

// Repr returns a string representation of the element, including its attributes and children.
func (e *ElementSpec) Repr() string {
	attributes := ""
	for k, v := range e.Attributes {
		attributes += fmt.Sprintf(" %s=\"%s\"", k, v)
	}
	attributes += fmt.Sprintf(" id=\"%s\"", e.Id)

	openingTag := fmt.Sprintf("<%s%s>", e.TagName, attributes)

	children := ""
	for _, child := range e.Children {
		children += child.Repr()
	}

	closingTag := fmt.Sprintf("</%s>", e.TagName)

	return fmt.Sprintf("%s%s%s%s", openingTag, e.Text, children, closingTag)
}

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
