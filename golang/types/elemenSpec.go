package types

import "fmt"

// ElementSpec represents the specification of an HTML element.
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

	childrenHTML := ""
	for _, child := range e.Children {
		childrenHTML += child.Repr()
	}

	closingTag := fmt.Sprintf("</%s>", e.TagName)

	return fmt.Sprintf("%s%s%s%s", openingTag, e.Text, childrenHTML, closingTag)
}
