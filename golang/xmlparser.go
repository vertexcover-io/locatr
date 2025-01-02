package locatr

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/antchfx/xmlquery"
)

var MAYBE_UNIQUE_XPATH_ATTRIBUTES = []string{
	"accessibility-id",
	"resource-id",
	"id",
	"name",
	"content-desc",
	"label",
	"value",
	"text",
}

func iterateNodes(node *xmlquery.Node, depth int) {
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
		iterateNodes(child, depth+1)
	}
}
func getElementHierarchyXpath(element *xmlquery.Node) string {
	var parts []string
	for element != nil {
		if element.Data == "AppiumAUT" {
			break
		}
		parent := element.Parent
		if parent != nil {
			var sibilingsOfSameTag []*xmlquery.Node
			for sib := parent.FirstChild; sib != nil; sib = sib.NextSibling {
				if sib.Type == xmlquery.ElementNode && sib.Data == element.Data {
					sibilingsOfSameTag = append(sibilingsOfSameTag, sib)
				}
			}
			if len(sibilingsOfSameTag) > 1 {
				pos := 1
				for i, sib := range sibilingsOfSameTag {
					if sib == element {
						pos = i + 1
						break
					}
				}
				parts = append(parts, fmt.Sprintf("%s[%d]", element.Data, pos))

			} else {
				parts = append(parts, element.Data)
			}
		} else {
			parts = append(parts, element.Data)
		}
		element = parent
	}
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return strings.Join(parts, "/")
}

func getElementLocatrs(element *xmlquery.Node) []string {
	locatrs := []string{}
	if element.Data == "hierarchy" {
		return locatrs
	}
	xpathStr := getElementHierarchyXpath(element)
	locatrs = append(locatrs, fmt.Sprintf("xpath,%s", xpathStr))
	for _, uniqueAttr := range MAYBE_UNIQUE_XPATH_ATTRIBUTES {
		if attrValue := element.SelectAttr(uniqueAttr); attrValue != "" {
			escapedValue := ""
			if strings.Contains(attrValue, "'") && !strings.Contains(attrValue, "\"") {
				escapedValue = fmt.Sprintf("\"%s\"", attrValue)
			} else if strings.Contains(attrValue, "\"") && !strings.Contains(attrValue, "'") {
				escapedValue = fmt.Sprintf("'%s'", attrValue)
			} else {
				escapedValue = strings.Replace(strings.Replace(attrValue, "\"", "\\\"", -1), "'", "\\'", -1)
				escapedValue = fmt.Sprintf("\"%s\"", escapedValue)
			}
			xpathStr := fmt.Sprintf("//%s[@%s=%s]", element.Data, uniqueAttr, escapedValue)
			isUnique, _ := isXpathUnique(xpathStr, element)
			if isUnique {
				locatrs = append(locatrs, fmt.Sprintf("xpath,%s", xpathStr))
				break
			}
		}
	}
	if len(locatrs) > 1 {
		return locatrs
	}
	xpathConditions := []string{}
	for _, uniqueAttr := range MAYBE_UNIQUE_XPATH_ATTRIBUTES {
		if attrValue := element.SelectAttr(uniqueAttr); attrValue != "" {
			escapedValue, _ := json.Marshal(attrValue)

			escapedValueStr := string(escapedValue)

			xpathConditions = append(xpathConditions, fmt.Sprintf("@%s=%s", uniqueAttr, escapedValueStr))

			xPathPredicate := strings.Join(xpathConditions, " and ")
			xPathStr := fmt.Sprintf("//%s[%s]", element.Data, xPathPredicate)
			isUnique, _ := isXpathUnique(xPathStr, element)
			if isUnique {
				locatrs = append(locatrs, fmt.Sprintf("xpath,%s", xPathStr))
				break
			}
		}

	}
	if len(locatrs) > 1 {
		return locatrs
	}
	baseXpath := getElementHierarchyXpath(element)
	_, elementIndx := isXpathUnique(baseXpath, element)
	if elementIndx == -1 {
		return locatrs
	} else if elementIndx != 0 {
		locatrs = append(locatrs, fmt.Sprintf("xpath,%s,[%d]", baseXpath, elementIndx+1))
	}
	return locatrs
}

func isElementVisible(
	element *xmlquery.Node, platform string,
) bool {
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
		if width > 0 && height > 0 {
			return true
		} else {
			return false
		}
	} else {
		visible := strings.ToLower(element.SelectAttr("visible"))
		if visible == "" {
			visible = "true"
		}
		return visible == "true"
	}
}

func getVisibleText(
	element *xmlquery.Node, platform string,
) string {
	if platform == "android" {
		return strings.TrimSpace(element.SelectAttr("text"))
	} else {
		if labelText := strings.TrimSpace(element.SelectAttr("lable")); labelText != "" {
			return labelText
		} else {
			return strings.TrimSpace(element.SelectAttr("value"))
		}
	}
}

func isValidElement(
	element *xmlquery.Node, platform string,
) bool {
	if element.Type == xmlquery.TextNode && strings.TrimSpace(element.Data) == "" {
		return false
	}

	if element.Data == "hierarchy" {
		return true
	}
	visible := isElementVisible(element, platform)
	if !visible {
		fmt.Println("element not visible", element, element.Data)
		return false
	}
	return true
}

func isXpathUnique(xPath string, element *xmlquery.Node) (bool, int) {
	root := element
	for root.Parent != nil {
		root = root.Parent
	}

	allElements := xmlquery.Find(root.FirstChild.NextSibling.NextSibling, xPath)
	elemLen := len(allElements)
	if elemLen == 1 {
		return true, -1
	}
	if elemLen == 0 {
		return false, -1
	}
	for indx, e := range allElements {
		if e == element {
			return false, indx
		}
	}
	return false, -1
}

func generateUniqueId(id string) string {
	md5Hash := md5.Sum([]byte(id))
	return string(md5Hash[:8])
}
func attrsToMap(attrs []xmlquery.Attr) map[string]string {
	attrMap := make(map[string]string)
	for _, attr := range attrs {
		attrMap[attr.Name.Local] = attr.Value
	}
	return attrMap
}
func printLocatrs(locatrs []string) {
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
) (*ElementSpec, error) {
	if !isValidElement(element, platform) {
		return nil, fmt.Errorf("not a valid element")
	}
	text := getVisibleText(element, platform)
	locatrs := getElementLocatrs(element)
	printLocatrs(locatrs)
	uniqueId := ""
	if len(locatrs) > 0 {
		uniqueId = generateUniqueId(locatrs[0])
	}
	children := []ElementSpec{}
	for child := element.FirstChild; child != nil; child = child.NextSibling {
		c, err := createElementSpec(child, root, platform)
		if err != nil && c != nil {
			children = append(children, *c)
		}

	}
	return &ElementSpec{
		TagName:    element.Data,
		ID:         uniqueId,
		Attributes: attrsToMap(element.Attr),
		Text:       text,
		Children:   children,
	}, nil
}

func buildElementTree(root *xmlquery.Node, paltform string) (*ElementSpec, error) {
	elementSpec, err := createElementSpec(root, root, paltform)
	if err != nil {
		return nil, err
	}
	return elementSpec, nil
}

func minifySource(source string, platform string) (*ElementSpec, error) {
	root, error := xmlquery.Parse(strings.NewReader(source))
	if error != nil {
		return nil, error
	}
	spec, error := buildElementTree(root.FirstChild.NextSibling.NextSibling, platform)
	if error != nil {
		return nil, error
	}
	return spec, nil
}