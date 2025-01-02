package locatr

import (
	"testing"
)

func TestContentStr(t *testing.T) {
	tests := []struct {
		name          string
		element       ElementSpec
		expected      string
		otherExpected string
	}{
		{
			name: "Single element with no attributes or children",
			element: ElementSpec{
				ID:      "1",
				TagName: "div",
				Text:    "Hello World",
			},
			expected: `<div id="1">Hello World</div>`,
		},
		{
			name: "Element with attributes",
			element: ElementSpec{
				ID:      "2",
				TagName: "span",
				Text:    "Hello",
				Attributes: map[string]string{
					"class": "greeting",
					"style": "color: red;",
				},
			},
			expected:      `<span class="greeting" style="color: red;" id="2">Hello</span>`,
			otherExpected: `<span style="color: red;" class="greeting" id="2">Hello</span>`,
		},
		{
			name: "Element with children",
			element: ElementSpec{
				ID:      "3",
				TagName: "ul",
				Children: []ElementSpec{
					{
						ID:      "4",
						TagName: "li",
						Text:    "Item 1",
					},
					{
						ID:      "5",
						TagName: "li",
						Text:    "Item 2",
					},
				},
			},
			expected: `<ul id="3"><li id="4">Item 1</li><li id="5">Item 2</li></ul>`,
		},
		{
			name: "Element with attributes and children",
			element: ElementSpec{
				ID:      "6",
				TagName: "div",
				Attributes: map[string]string{
					"class": "container",
				},
				Children: []ElementSpec{
					{
						ID:      "7",
						TagName: "p",
						Text:    "Paragraph",
					},
				},
			},
			expected: `<div class="container" id="6"><p id="7">Paragraph</p></div>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.element.ContentStr()
			if result != tt.expected && result != tt.otherExpected {
				t.Errorf("got %s, want %s or %s", result, tt.expected, tt.otherExpected)
			}
		})
	}
}
