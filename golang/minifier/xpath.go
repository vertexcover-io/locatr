package minifier

import (
	"fmt"
	"strings"
)

type Document interface {
	Find(xpath string) []Node
	IndexOf(node Node) int
}

type Node interface {
	TagName() string
	IsElement() bool
	HasParent() bool
	GetAttribute(key string) string
	GetParent() Node
	ChildNodes() []Node
	Index() int
}

var (
	// Attributes on nodes that are likely to be unique to the node. These are considered in order
	unique_xpath_attrs = []string{"name", "content-desc", "id", "resource-id", "accessibility-id"}

	// Attributes that are recommended as fallback but ideally only in conjunction with other
	// attributes
	maybe_unique_xpath_attrs = []string{"label", "text", "value"}
)

func GetOptimalXPath(doc Document, domNode Node) string {
	// BASE CASE #1: If this isn't an element, we're above the root, return empty string
	if domNode == nil || domNode.TagName() == "" || !domNode.IsElement() {
		return ""
	}

	attrPairs := generateAttrPairs()

	cases := [][]string{
		// BASE CASE #2: If the node has a unique attribute or content attribute, return an absolute
		// XPath with that attribute
		unique_xpath_attrs,

		// BASE CASE #3: If the node has a unique pair of attributes including 'maybe' attributes,
		// return an XPATH based on that pair
		attrPairs,

		// BASE CASE #4: Look for 'maybe' unique attributes on its own. It's better than if we find one
		// of these that's unique in conjunction with another attribute, but if not, it is still better
		// than hierarchial query.
		maybe_unique_xpath_attrs,

		// BASE CASE #5: Look to see if the node type is unique in the document
		{},
	}

	// It's possible that in all of these cases we don't find a truly unique selector. But a selector
	// qualified by attribute with an index attached, like //*[@id="foo"][1], which is still better
	// than a fully path-based selector.
	var semiUniqueXpath string

	for _, attrCase := range cases {
		ok, isUnique, xpath := getUniqueXPATH(doc, domNode, attrCase)
		if !ok {
			continue
		}

		if isUnique {
			return xpath
		} else if semiUniqueXpath == "" {
			semiUniqueXpath = xpath
		}
	}

	// once we have gone through all our cases, if we do still have a semi unique xpath, send that back
	if semiUniqueXpath != "" {
		return semiUniqueXpath
	}

	// otherwise fall back to a purely hierarchial expression of this dom node's position in the
	// document as a last resort.
	// First get the relative xpath of this node using tagname
	xpath := fmt.Sprintf("/%s", domNode.TagName())

	// if this node has siblings of the same tagname, get the index of this node
	if domNode.HasParent() {
		var siblings []Node
		for _, child := range domNode.GetParent().ChildNodes() {
			if child.IsElement() && child.TagName() == domNode.TagName() {
				siblings = append(siblings, child)
			}
		}

		// If there's more than one sibling, append the index
		if len(siblings) > 1 {
			idx := domNode.Index()

			xpath = fmt.Sprintf("%s[%d]", xpath, idx)
		}

	}

	// Make a recursive call to this nodes parents and preprend it to this xpath
	parentXPath := GetOptimalXPath(doc, domNode.GetParent())
	return parentXPath + xpath
}

func generateAttrPairs() []string {
	var attrsForPairs []string
	attrsForPairs = append(attrsForPairs, unique_xpath_attrs...)
	attrsForPairs = append(attrsForPairs, maybe_unique_xpath_attrs...)

	var attrPairs []string
	for i, attr := range attrsForPairs {
		for j := i + 1; j < len(attrsForPairs); j += 1 {
			pair := fmt.Sprintf("%s %s", attr, attrsForPairs[j])
			attrPairs = append(attrPairs, pair)
		}
	}

	return attrPairs
}

func getUniqueXPATH(doc Document, domNode Node, attrs []string) (valid bool, unique bool, xpath string) {
	isNodeName := len(attrs) == 0
	if isNodeName {
		xpath := fmt.Sprintf("//%s", domNode.TagName())

		isUnique, _ := determineXpathUniqueness(xpath, doc, domNode)
		if isUnique {
			if !domNode.HasParent() {
				xpath = fmt.Sprintf("/%s", domNode.TagName())
			}
			return true, true, xpath
		}
		return false, false, ""
	}

	var uniqueXPath string
	var semiUniqueXPath string

	tagForXpath := domNode.TagName()
	if tagForXpath == "" {
		tagForXpath = "*"
	}
	isPair := len(strings.Fields(attrs[0])) > 1

	for _, attr := range attrs {
		var xpath string

		if isPair {
			attr1, attr2, ok := strings.Cut(attr, " ")
			if !ok {
				panic("generateUniqueXPATH invalid state")
			}

			attr1Value, attr2Value := domNode.GetAttribute(attr1), domNode.GetAttribute(attr2)
			if attr1Value == "" || attr2Value == "" {
				continue
			}

			xpath = fmt.Sprintf(
				"//%s[@%s=\"%s\" and @%s=\"%s\"]",
				tagForXpath,
				attr1, attr1Value,
				attr2, attr2Value,
			)
		} else {
			attrValue := domNode.GetAttribute(attr)
			if attrValue == "" {
				continue
			}
			xpath = fmt.Sprintf(
				"//%s[@%s=\"%s\"]",
				tagForXpath,
				attr, attrValue,
			)
		}

		isUnique, idx := determineXpathUniqueness(xpath, doc, domNode)
		if isUnique {
			uniqueXPath = xpath
			break
		}

		if semiUniqueXPath == "" {
			semiUniqueXPath = fmt.Sprintf("(%s)[%d]", xpath, idx+1)
		}
	}

	if uniqueXPath != "" {
		return true, true, uniqueXPath
	}
	if semiUniqueXPath != "" {
		return true, false, semiUniqueXPath
	}
	return false, false, ""
}

func determineXpathUniqueness(xpath string, doc Document, domNode Node) (bool, int) {
	elems := doc.Find(xpath)
	if len(elems) > 1 {
		return false, doc.IndexOf(domNode)
	}
	return true, 0
}
