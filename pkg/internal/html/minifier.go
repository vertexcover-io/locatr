package html

import (
	"fmt"
	"strings"

	"github.com/antchfx/htmlquery"
	"github.com/vertexcover-io/locatr/pkg/internal/utils"
	"github.com/vertexcover-io/locatr/pkg/types"
	"golang.org/x/net/html"
)

// nolint:unused
func PrintXmlTree(node *html.Node, depth int) {
	if node == nil {
		return
	}
	if node.Type == html.TextNode && strings.TrimSpace(node.Data) == "" {
		return
	}

	fmt.Printf("%sNode: %s", strings.Repeat("  ", depth), node.Data)
	if len(node.Attr) > 0 {
		fmt.Print(" [Attributes: ")
		for _, attr := range node.Attr {
			fmt.Printf("%s=%q ", attr.Key, attr.Val)
		}
		fmt.Print("]")
	}
	fmt.Println()

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		PrintXmlTree(child, depth+1)
	}
}

func findFirstElementNode(node *html.Node) *html.Node {
	// If the current node is an element node, return it immediately
	if node.Type == html.ElementNode {
		return node
	}

	// Recursively search through child nodes
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		// Recursively call findFirstElementNode on each child
		found := findFirstElementNode(child)
		// If an element node is found, return it
		if found != nil {
			return found
		}
	}

	// If no element node is found, return nil
	return nil
}

// For HTML, unless we evaluate CSS as well, we can never be certain if the
// element is visible or not. However, we eliminate the base cases that
// is possible with html only.
func isElementVisible(element *html.Node) bool {
	// 1. Skip non-element Nodes
	if element.Type != html.ElementNode {
		return false
	}

	// 2. Tags that never render visible content
	switch strings.ToLower(element.Data) {
	case "script", "style", "template", "noscript", "head", "meta", "link":
		return false
	}

	// 3. Check if element has hidden attribute
	if hasAttr(element, "hidden") {
		return false
	}

	// 4. Check if aria hidden has been applied
	if val, ok := attrVal(element, "aria-hidden"); ok && strings.EqualFold(val, "true") {
		return false
	}

	// 5. Check if element is hidden with inline-styles
	if style, ok := attrVal(element, "style"); ok {
		s := strings.ToLower(style)
		if strings.Contains(s, "display:none") ||
			strings.Contains(s, "visibility:hidden") ||
			strings.Contains(s, "opacity: 0") {
			return false
		}
	}

	return true
}

func hasAttr(element *html.Node, name string) bool {
	_, ok := attrVal(element, name)
	return ok
}

func attrVal(element *html.Node, name string) (string, bool) {
	for _, a := range element.Attr {
		if strings.EqualFold(a.Key, name) {
			return a.Val, true
		}
	}
	return "", false
}

func escapeString(str string) string {
	return html.EscapeString(str)
}

func getVisibleText(element *html.Node) string {
	txt := element.Data
	return escapeString(strings.TrimSpace(txt))
}

func isElementValid(element *html.Node) bool {
	if element.Type == html.TextNode && strings.TrimSpace(element.Data) == "" {
		return false
	}
	if element.Data == "hierarchy" {
		return true
	}
	// this check is essential, in iOS, there are cases where the parent heirarchy is marked as
	// not visible, despite having children as visible. In case of iOS, we can't trust on
	// element visibility.
	if element.FirstChild != nil {
		return true
	}
	visible := isElementVisible(element)
	return visible
}

func attrsToMap(attrs []html.Attribute) map[string]string {
	attrMap := make(map[string]string)
	for _, attr := range attrs {
		attrMap[attr.Key] = escapeString(attr.Val)
	}
	return attrMap
}

// nolint:unused
func PrintLocatrs(locatrs []string) {
	fmt.Printf("[")
	for i, l := range locatrs {
		if i == len(locatrs)-1 {
			fmt.Printf("'%s'", l)
			continue
		}
		fmt.Printf("'%s', ", l)

	}
	fmt.Println("]")

}

func createElementSpec(element *html.Node, root *html.Node) (*types.ElementSpec, error) {
	if !isElementValid(element) {
		return nil, fmt.Errorf("not a valid element")
	}

	text := getVisibleText(element)
	doc := NewHTMLDoc(root)
	node := NewHTMLNode(element)
	xpath := GetOptimalXPath(doc, node)
	uniqueId := utils.GenerateUniqueId(xpath)

	children := []types.ElementSpec{}
	for child := element.FirstChild; child != nil; child = child.NextSibling {
		c, err := createElementSpec(child, root)
		if err == nil && c != nil {
			children = append(children, *c)
		}
	}
	return &types.ElementSpec{
		TagName:    element.Data,
		Id:         uniqueId,
		Attributes: attrsToMap(element.Attr),
		Text:       text,
		Children:   children,
	}, nil
}

func MinifySource(source string) (*types.ElementSpec, error) {
	if source == "" {
		return nil, fmt.Errorf("source is empty")
	}
	root, err := htmlquery.Parse(strings.NewReader(source))
	if err != nil {
		return nil, err
	}
	node := findFirstElementNode(root)
	spec, err := createElementSpec(node, node)
	if err != nil {
		return nil, err
	}
	return spec, nil
}

func CreateLocatorMap(source string) (map[string][]string, error) {
	if source == "" {
		return nil, fmt.Errorf("source is empty")
	}
	root, err := htmlquery.Parse(strings.NewReader(source))
	if err != nil {
		return nil, err
	}
	elementMap := make(map[string][]string)

	var processElement func(*html.Node)

	doc := NewHTMLDoc(root)
	processElement = func(elem *html.Node) {
		node := NewHTMLNode(elem)
		xpath := GetOptimalXPath(doc, node)
		if xpath != "" {
			uniqueId := utils.GenerateUniqueId(xpath)
			elementMap[uniqueId] = []string{xpath}
		}

		for child := elem.FirstChild; child != nil; child = child.NextSibling {
			if isElementValid(child) {
				processElement(child)
			}
		}
	}
	processElement(findFirstElementNode(root))
	return elementMap, nil
}
