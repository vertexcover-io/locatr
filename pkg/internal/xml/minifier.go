package xml

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"

	"github.com/antchfx/xmlquery"
	"github.com/vertexcover-io/locatr/pkg/internal/utils"
	"github.com/vertexcover-io/locatr/pkg/types"
)

// nolint:unused
func PrintXmlTree(node *xmlquery.Node, depth int) {
	if node == nil {
		return
	}
	if node.Type == xmlquery.TextNode && strings.TrimSpace(node.Data) == "" {
		return
	}

	fmt.Printf("%sNode: %s", strings.Repeat("  ", depth), node.Data)
	if len(node.Attr) > 0 {
		fmt.Print(" [Attributes: ")
		for _, attr := range node.Attr {
			fmt.Printf("%s=%q ", attr.Name.Local, attr.Value)
		}
		fmt.Print("]")
	}
	fmt.Println()

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		PrintXmlTree(child, depth+1)
	}
}

func findFirstElementNode(node *xmlquery.Node) *xmlquery.Node {
	// If the current node is an element node, return it immediately
	if node.Type == xmlquery.ElementNode {
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

func isElementVisible(element *xmlquery.Node, platform string) bool {
	// Android visibility check using bounds
	if platform == "android" {
		bounds := element.SelectAttr("bounds")
		if bounds == "" {
			return false
		}

		boundsParts := strings.Split(bounds, "][")
		if len(boundsParts) != 2 {
			return false
		}

		start := strings.Split(strings.Replace(boundsParts[0], "[", "", -1), ",")
		end := strings.Split(strings.Replace(boundsParts[1], "]", "", -1), ",")

		wstartInt, _ := strconv.Atoi(start[0])
		wendInt, _ := strconv.Atoi(end[0])
		width := wendInt - wstartInt

		hstartInt, _ := strconv.Atoi(start[1])
		hendInt, _ := strconv.Atoi(end[1])
		height := hendInt - hstartInt

		return width > 0 && height > 0
	}

	// iOS visibility check
	visible := strings.ToLower(element.SelectAttr("visible"))
	return visible == "true" || visible == ""
}

func escapeString(str string) string {
	var buf bytes.Buffer
	err := xml.EscapeText(&buf, []byte(str))
	if err != nil {
		return str
	}
	return buf.String()
}

func getVisibleText(
	element *xmlquery.Node, platform string,
) string {
	if platform == "android" {
		return escapeString(strings.TrimSpace(element.SelectAttr("text")))
	} else {
		if labelText := strings.TrimSpace(element.SelectAttr("label")); labelText != "" {
			return escapeString(labelText)
		} else {
			return escapeString(strings.TrimSpace(element.SelectAttr("value")))
		}
	}
}

func isElementValid(
	element *xmlquery.Node, platform string,
) bool {
	if element.Type == xmlquery.TextNode && strings.TrimSpace(element.Data) == "" {
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
	visible := isElementVisible(element, platform)
	return visible
}

func attrsToMap(attrs []xmlquery.Attr) map[string]string {
	attrMap := make(map[string]string)
	for _, attr := range attrs {
		attrMap[attr.Name.Local] = escapeString(attr.Value)
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

func createElementSpec(
	element *xmlquery.Node, root *xmlquery.Node, platform string,
) (*types.ElementSpec, error) {
	if !isElementValid(element, platform) {
		return nil, fmt.Errorf("not a valid element")
	}
	text := getVisibleText(element, platform)
	doc := NewXMLDoc(root)
	node := NewXMLNode(element)
	xpath := GetOptimalXPath(doc, node)
	uniqueId := utils.GenerateUniqueId(xpath)

	children := []types.ElementSpec{}
	for child := element.FirstChild; child != nil; child = child.NextSibling {
		c, err := createElementSpec(child, root, platform)
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

func MinifySource(source string, platform string) (*types.ElementSpec, error) {
	if source == "" {
		return nil, fmt.Errorf("source is empty")
	}
	root, err := xmlquery.Parse(strings.NewReader(source))
	if err != nil {
		return nil, err
	}
	node := findFirstElementNode(root)
	spec, err := createElementSpec(node, node, platform)
	if err != nil {
		return nil, err
	}
	return spec, nil
}

func CreateLocatorMap(source string, platform string) (map[string][]string, error) {
	if source == "" {
		return nil, fmt.Errorf("source is empty")
	}
	root, err := xmlquery.Parse(strings.NewReader(source))
	if err != nil {
		return nil, err
	}
	elementMap := make(map[string][]string)

	var processElement func(*xmlquery.Node)

	doc := NewXMLDoc(root)
	processElement = func(elem *xmlquery.Node) {
		node := NewXMLNode(elem)
		xpath := GetOptimalXPath(doc, node)
		if xpath != "" {
			uniqueId := utils.GenerateUniqueId(xpath)
			elementMap[uniqueId] = []string{xpath}
		}

		for child := elem.FirstChild; child != nil; child = child.NextSibling {
			if isElementValid(child, platform) {
				processElement(child)
			}
		}
	}
	processElement(findFirstElementNode(root))
	return elementMap, nil
}
