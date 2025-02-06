package appiumLocatr

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/antchfx/xmlquery"
	"github.com/vertexcover-io/locatr/golang/elementSpec"
)

func TestFindFirstElementNode(t *testing.T) {
	// Test Case 1: Basic Element Node
	func() {
		xmlStr := `<root><child>Text</child></root>`
		doc, _ := xmlquery.Parse(strings.NewReader(xmlStr))
		result := findFirstElementNode(doc)

		if result == nil {
			t.Errorf("Expected to find an element node, got nil")
		}
		if result.Data != "root" {
			t.Errorf("Expected 'root' element, got %s", result.Data)
		}
	}()

	// Test Case 2: Nested Element Nodes
	func() {
		xmlStr := `<root><intermediate><deep>Content</deep></intermediate></root>`
		doc, _ := xmlquery.Parse(strings.NewReader(xmlStr))
		result := findFirstElementNode(doc)

		if result == nil {
			t.Errorf("Expected to find an element node, got nil")
		}
		if result.Data != "root" {
			t.Errorf("Expected 'root' element, got %s", result.Data)
		}
	}()

	// Test Case 3: Multiple Text Nodes
	func() {
		xmlStr := `<root>Some text<child>More text</child>Another text</root>`
		doc, _ := xmlquery.Parse(strings.NewReader(xmlStr))
		result := findFirstElementNode(doc)

		if result == nil {
			t.Errorf("Expected to find an element node, got nil")
		}
		if result.Data != "root" {
			t.Errorf("Expected 'root' element, got %s", result.Data)
		}
	}()

	// Test Case 4: Empty Document
	func() {
		xmlStr := ``
		doc, _ := xmlquery.Parse(strings.NewReader(xmlStr))
		result := findFirstElementNode(doc)

		if result != nil {
			t.Errorf("Expected nil for empty document, got %v", result)
		}
	}()

	// Test Case 5: Document with Only Text Nodes
	func() {
		xmlStr := `Some random text more text`
		doc, _ := xmlquery.Parse(strings.NewReader(xmlStr))
		result := findFirstElementNode(doc)

		if result != nil {
			t.Errorf("Expected nil for text-only document, got %v", result)
		}
	}()

	// Test Case 6: Complex Nested Structure
	func() {
		xmlStr := `
        <root>
            <level1>
                <level2>
                    <level3>Deep content</level3>
                </level2>
            </level1>
        </root>`
		doc, _ := xmlquery.Parse(strings.NewReader(xmlStr))
		result := findFirstElementNode(doc)

		if result == nil {
			t.Errorf("Expected to find an element node, got nil")
		}
		if result.Data != "root" {
			t.Errorf("Expected 'root' element, got %s", result.Data)
		}
	}()
}

func TestGetElementHierarchyXpath(t *testing.T) {
	// Helper function to parse XML and get a specific node
	getNodeByPath := func(xmlStr, path string) *xmlquery.Node {
		doc, _ := xmlquery.Parse(strings.NewReader(xmlStr))
		nodes := xmlquery.Find(doc, path)
		if len(nodes) == 0 {
			t.Fatalf("No node found for path %s", path)
		}
		return nodes[0]
	}

	// Test Case 1: Simple nested structure
	func() {
		xmlStr := `
        <root>
            <parent>
                <child>Content</child>
            </parent>
        </root>`
		childNode := getNodeByPath(xmlStr, "//child")
		xpath := getElementHierarchyXpath(childNode)

		expectedXPath := "/root/parent/child"
		if xpath != expectedXPath {
			t.Errorf("Expected %s, got %s", expectedXPath, xpath)
		}
	}()

	// Test Case 2: Multiple siblings with same tag
	func() {
		xmlStr := `
        <root>
            <parent>
                <child>First</child>
                <child>Second</child>
                <child>Third</child>
            </parent>
        </root>`
		secondChildNode := getNodeByPath(xmlStr, "//child[2]")
		xpath := getElementHierarchyXpath(secondChildNode)

		expectedXPath := "/root/parent/child[2]"
		if xpath != expectedXPath {
			t.Errorf("Expected %s, got %s", expectedXPath, xpath)
		}
	}()

	// Test Case 3: Deep nested structure
	func() {
		xmlStr := `
        <root>
            <level1>
                <level2>
                    <level3>Deep Content</level3>
                </level2>
            </level1>
        </root>`
		deepNode := getNodeByPath(xmlStr, "//level3")
		xpath := getElementHierarchyXpath(deepNode)

		expectedXPath := "/root/level1/level2/level3"
		if xpath != expectedXPath {
			t.Errorf("Expected %s, got %s", expectedXPath, xpath)
		}
	}()

	// Test Case 4: Mixed siblings
	func() {
		xmlStr := `
        <root>
            <parent>
                <div>Div 1</div>
                <child>First Child</child>
                <div>Div 2</div>
                <child>Second Child</child>
            </parent>
        </root>`
		secondChildNode := getNodeByPath(xmlStr, "//child[2]")
		xpath := getElementHierarchyXpath(secondChildNode)

		expectedXPath := "/root/parent/child[2]"
		if xpath != expectedXPath {
			t.Errorf("Expected %s, got %s", expectedXPath, xpath)
		}
	}()

	// Test Case 5: Stop at AppiumAUT
	func() {
		xmlStr := `
        <AppiumAUT>
            <root>
                <child>Stop here</child>
            </root>
        </AppiumAUT>`
		childNode := getNodeByPath(xmlStr, "//child")
		xpath := getElementHierarchyXpath(childNode)

		expectedXPath := "/root/child"
		if xpath != expectedXPath {
			t.Errorf("Expected /root/child string, got %s", xpath)
		}
	}()
}

func TestGetElementLocatrs(t *testing.T) {
	// Helper function to parse XML and get a specific node
	getNodeByPath := func(xmlStr, path string) *xmlquery.Node {
		doc, _ := xmlquery.Parse(strings.NewReader(xmlStr))
		nodes := xmlquery.Find(doc, path)
		if len(nodes) == 0 {
			t.Fatalf("No node found for path %s", path)
		}
		return nodes[0]
	}

	// Test Case 1: Simple Unique ID Attribute
	func() {
		xmlStr := `<root><element id="unique-id">Content</element></root>`
		node := getNodeByPath(xmlStr, "//element[@id='unique-id']")
		locatrs := getElementLocatrs(node)

		if len(locatrs) < 2 {
			t.Errorf("Expected multiple locators, got %v", locatrs)
		}

		found := false
		for _, locatr := range locatrs {
			if strings.Contains(locatr, "@id") {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Expected locator with ID attribute, got %v", locatrs)
		}
	}()

	// Test Case 2: Multiple Siblings
	func() {
		xmlStr := `
        <root>
            <parent>
                <child>First</child>
                <child>Second</child>
                <child>Third</child>
            </parent>
        </root>`
		secondChild := getNodeByPath(xmlStr, "//child[2]")
		locatrs := getElementLocatrs(secondChild)

		if len(locatrs) == 0 {
			t.Errorf("Expected locators, got empty list")
		}
	}()

	// Test Case 3: Complex Nested Structure with Attributes
	func() {
		xmlStr := `
        <root>
            <parent resource-id="parent-container">
                <child content-desc="unique-child">Specific Content</child>
            </parent>
        </root>`
		specificChild := getNodeByPath(xmlStr, "//child[@content-desc='unique-child']")
		locatrs := getElementLocatrs(specificChild)

		if len(locatrs) < 2 {
			t.Errorf("Expected multiple locators, got %v", locatrs)
		}

		hierarchyFound := false
		attributeFound := false
		for _, locatr := range locatrs {
			if strings.Contains(locatr, "/root/parent/child") {
				hierarchyFound = true
			}
			if strings.Contains(locatr, "@content-desc") {
				attributeFound = true
			}
		}

		if !hierarchyFound || !attributeFound {
			t.Errorf("Missing expected locator types in %v", locatrs)
		}
	}()

	// Test Case 4: No Unique Locators
	func() {
		xmlStr := `
        <root>
            <parent>
                <child>Generic Content</child>
                <child>Generic Content</child>
            </parent>
        </root>`
		secondChild := getNodeByPath(xmlStr, "//child[2]")
		locatrs := getElementLocatrs(secondChild)

		if len(locatrs) == 0 {
			t.Errorf("Expected locators even without unique attributes")
		}

		// Verify index-based locator
		indexLocatorFound := false
		for _, locatr := range locatrs {
			if strings.Contains(locatr, "[2]") {
				indexLocatorFound = true
				break
			}
		}

		if !indexLocatorFound {
			t.Errorf("Expected index-based locator, got %v", locatrs)
		}
	}()

	// Test Case 5: Hierarchy Locator with Index
	func() {
		xmlStr := `
        <root>
            <level1>
                <level2>
                    <child>First</child>
                    <child>Second</child>
                </level2>
            </level1>
        </root>`
		secondChild := getNodeByPath(xmlStr, "//child[2]")
		locatrs := getElementLocatrs(secondChild)

		indexLocatorFound := false
		for _, locatr := range locatrs {
			if strings.Contains(locatr, "[2]") {
				indexLocatorFound = true
				break
			}
		}

		if !indexLocatorFound {
			t.Errorf("Expected index-based locator, got %v", locatrs)
		}
	}()
}

func TestIsElementVisible(t *testing.T) {
	// Test Case 1: Android Visible Element
	func() {
		xmlStr := `<element bounds="[0,0][100,100]"/>`
		element, _ := xmlquery.Parse(strings.NewReader(xmlStr))
		result := isElementVisible(element.FirstChild.NextSibling, "android")

		if !result {
			t.Errorf("Expected visible element, got false")
		}
	}()

	// Test Case 2: Android Zero Width/Height
	func() {
		xmlStr := `<element bounds="[0,0][0,0]"/>`
		element, _ := xmlquery.Parse(strings.NewReader(xmlStr))
		result := isElementVisible(element.FirstChild.NextSibling, "android")

		if result {
			t.Errorf("Expected invisible element, got true")
		}
	}()

	// Test Case 3: Android Invalid Bounds
	func() {
		xmlStr := `<element bounds="invalid"/>`
		element, _ := xmlquery.Parse(strings.NewReader(xmlStr))
		result := isElementVisible(element.FirstChild.NextSibling, "android")

		if result {
			t.Errorf("Expected false for invalid bounds, got true")
		}
	}()

	// Test Case 4: iOS Visible Element
	func() {
		xmlStr := `<element visible="true"/>`
		element, _ := xmlquery.Parse(strings.NewReader(xmlStr))
		result := isElementVisible(element.FirstChild.NextSibling, "ios")

		if !result {
			t.Errorf("Expected visible element, got false")
		}
	}()

	// Test Case 5: iOS Invisible Element
	func() {
		xmlStr := `<element visible="false"/>`
		element, _ := xmlquery.Parse(strings.NewReader(xmlStr))
		result := isElementVisible(element.FirstChild.NextSibling, "ios")

		if result {
			t.Errorf("Expected invisible element, got true")
		}
	}()

	// Test Case 6: iOS No Visibility Attribute
	func() {
		xmlStr := `<element/>`
		element, _ := xmlquery.Parse(strings.NewReader(xmlStr))
		result := isElementVisible(element.FirstChild.NextSibling, "ios")

		if !result {
			t.Errorf("Expected default visible, got false")
		}
	}()
}

func TestGetVisibleText(t *testing.T) {
	// Android Text Attribute
	func() {
		xmlStr := `<element text="Android Text"/>`
		element, _ := xmlquery.Parse(strings.NewReader(xmlStr))
		text := getVisibleText(element.FirstChild.NextSibling, "android")

		if text != "Android Text" {
			t.Errorf("Expected 'Android Text', got '%s'", text)
		}
	}()

	// iOS Label Attribute
	func() {
		xmlStr := `<element label="iOS Label"/>`
		element, _ := xmlquery.Parse(strings.NewReader(xmlStr))
		text := getVisibleText(element.FirstChild.NextSibling, "ios")

		if text != "iOS Label" {
			t.Errorf("Expected 'iOS Label', got '%s'", text)
		}
	}()

	// iOS Value Attribute Fallback
	func() {
		xmlStr := `<element value="iOS Value"/>`
		element, _ := xmlquery.Parse(strings.NewReader(xmlStr))
		text := getVisibleText(element.FirstChild.NextSibling, "ios")

		if text != "iOS Value" {
			t.Errorf("Expected 'iOS Value', got '%s'", text)
		}
	}()
}

func TestIsValidElement(t *testing.T) {
	// Empty Text Node
	func() {
		xmlStr := `<element> </element>`
		element, _ := xmlquery.Parse(strings.NewReader(xmlStr))
		valid := isValidElement(element.FirstChild.NextSibling, "android")

		if valid {
			t.Errorf("Expected invalid for empty text node")
		}
	}()

	// Hierarchy Node
	func() {
		xmlStr := `<hierarchy></hierarchy>`
		element, _ := xmlquery.Parse(strings.NewReader(xmlStr))
		valid := isValidElement(element.FirstChild.NextSibling, "android")

		if !valid {
			t.Errorf("Expected valid for hierarchy node")
		}
	}()

	// Visible Element
	func() {
		xmlStr := `<element bounds="[0,0][100,100]"/>`
		element, _ := xmlquery.Parse(strings.NewReader(xmlStr))
		valid := isValidElement(element.FirstChild.NextSibling, "android")

		if !valid {
			t.Errorf("Expected valid for visible element")
		}
	}()
}

// ----------------------------

func TestIsXpathUnique(t *testing.T) {
	tests := []struct {
		name          string
		xml           string
		xpath         string
		targetPath    string
		expectUnique  bool
		expectedIndex int
	}{
		{
			name:          "Single Element",
			xml:           `<root><element id="unique"/></root>`,
			xpath:         "//element[@id='unique']",
			targetPath:    "//element",
			expectUnique:  true,
			expectedIndex: -1,
		},
		{
			name: "Multiple Elements Same XPath",
			xml: `<root>
				<element id="same">First</element>
				<element id="same">Second</element>
			</root>`,
			xpath:         "//element[@id='same']",
			targetPath:    "//element[1]",
			expectUnique:  false,
			expectedIndex: 0,
		},
		{
			name:          "No Matching Elements",
			xml:           `<root><element id="exists"/></root>`,
			xpath:         "//element[@id='nonexistent']",
			targetPath:    "//element",
			expectUnique:  false,
			expectedIndex: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := xmlquery.Parse(strings.NewReader(tt.xml))
			if err != nil {
				t.Fatalf("Failed to parse XML: %v", err)
			}

			targetElement := xmlquery.FindOne(doc, tt.targetPath)
			if targetElement == nil {
				t.Fatal("Target element not found")
			}

			unique, index := isXpathUnique(tt.xpath, targetElement)
			if unique != tt.expectUnique {
				t.Errorf("Expected unique=%v, got %v", tt.expectUnique, unique)
			}
			if index != tt.expectedIndex {
				t.Errorf("Expected index=%d, got %d", tt.expectedIndex, index)
			}
		})
	}
}

func TestAttrsToMap(t *testing.T) {
	tests := []struct {
		name     string
		attrs    []xmlquery.Attr
		expected map[string]string
	}{
		{
			name:     "Empty Attributes",
			attrs:    []xmlquery.Attr{},
			expected: map[string]string{},
		},
		{
			name: "Single Attribute",
			attrs: []xmlquery.Attr{
				{Name: xml.Name{Local: "id"}, Value: "test"},
			},
			expected: map[string]string{"id": "test"},
		},
		{
			name: "Multiple Attributes",
			attrs: []xmlquery.Attr{
				{Name: xml.Name{Local: "id"}, Value: "test"},
				{Name: xml.Name{Local: "class"}, Value: "container"},
			},
			expected: map[string]string{
				"id":    "test",
				"class": "container",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := attrsToMap(tt.attrs)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected map length %d, got %d", len(tt.expected), len(result))
			}

			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("Expected %s=%s, got %s=%s", k, v, k, result[k])
				}
			}
		})
	}
}

func validateElementSpec(t *testing.T, spec *elementSpec.ElementSpec) {
	if spec == nil {
		t.Error("Nil ElementSpec provided to validateElementSpec")
		return
	}

	if spec.TagName == "" {
		t.Error("Expected non-empty TagName")
	}
}

func TestMinifySource(t *testing.T) {
	tests := []struct {
		name     string
		xml      string
		platform string
		wantErr  bool
	}{
		{
			name: "Valid Source",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<hierarchy>
				<android.widget.Button bounds="[0,0][100,100]" text="Click me"/>
			</hierarchy>`,
			platform: "android",
			wantErr:  false,
		},
		{
			name:     "Empty Source",
			xml:      "",
			platform: "android",
			wantErr:  true,
		},
		{
			name:     "Invalid XML",
			xml:      "<unclosed>",
			platform: "android",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.xml == "" {
				_, err := minifySource(tt.xml, tt.platform)
				if err == nil {
					t.Error("Expected error for empty source")
				}
				return
			}

			spec, err := minifySource(tt.xml, tt.platform)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if spec == nil {
				t.Error("Expected non-nil ElementSpec")
				return
			}

			validateElementSpec(t, spec)
		})
	}
}

func TestMapElementsToJson(t *testing.T) {
	tests := []struct {
		name          string
		xml           string
		platform      string
		wantErr       bool
		expectedCount int
	}{
		{
			name:          "Single Element",
			xml:           `<hierarchy><element bounds="[0,0][100,100]" resource-id="unique"/></hierarchy>`,
			platform:      "android",
			wantErr:       false,
			expectedCount: 1,
		},
		{
			name:          "Empty Source",
			xml:           "",
			platform:      "android",
			wantErr:       true,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.xml == "" {
				_, err := mapElementsToJson(tt.xml, tt.platform)
				if err == nil {
					t.Error("Expected error for empty source")
				}
				return
			}

			elementMap, err := mapElementsToJson(tt.xml, tt.platform)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if elementMap == nil {
				t.Error("Expected non-nil elementMap")
				return
			}

			if len(*elementMap) != tt.expectedCount {
				t.Errorf("Expected %d elements, got %d", tt.expectedCount, len(*elementMap))
			}
		})
	}
}
