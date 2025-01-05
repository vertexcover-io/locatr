package elementSpec

import (
	"fmt"

	"gopkg.in/validator.v2"
)

type ElementSpec struct {
	ID         string            `json:"id" validate:"required"`
	TagName    string            `json:"tag_name" validate:"required"`
	Text       string            `json:"text"`
	Attributes map[string]string `json:"attributes"`
	Children   []ElementSpec     `json:"children"`
}

type IdToLocatorMap map[string][]string

func (e *ElementSpec) Validate() error {
	if errs := validator.Validate(e); errs != nil {
		return errs
	}
	return nil
}

func (e *ElementSpec) ContentStr() string {
	attributes := ""
	for k, v := range e.Attributes {
		attributes += fmt.Sprintf(" %s=\"%s\"", k, v)
	}
	attributes += fmt.Sprintf(" id=\"%s\"", e.ID)

	openingTag := fmt.Sprintf("<%s%s>", e.TagName, attributes)
	closingTag := fmt.Sprintf("</%s>", e.TagName)

	childrenHTML := ""
	for _, child := range e.Children {
		childrenHTML += child.ContentStr()
	}
	return fmt.Sprintf("%s%s%s%s", openingTag, e.Text, childrenHTML, closingTag)
}
