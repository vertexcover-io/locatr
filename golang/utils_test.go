package locatr

import (
	"reflect"
	"testing"
)

func TestGetUniqueStringArray(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "No duplicates",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "With duplicates",
			input:    []string{"a", "b", "a", "c", "b"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "All duplicates",
			input:    []string{"a", "a", "a"},
			expected: []string{"a"},
		},
		{
			name:     "Empty array",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "Single element",
			input:    []string{"a"},
			expected: []string{"a"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetUniqueStringArray(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFixLLmJson(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "No formatting",
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "Surrounded by backticks",
			input:    "```{\"key\": \"value\"}```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "Prefixed with json and surrounded by backticks",
			input:    "```json{\"key\": \"value\"}```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "Prefixed with json without backticks",
			input:    "json{\"key\": \"value\"}",
			expected: `{"key": "value"}`,
		},
		{
			name:     "Backticks only at start",
			input:    "```{\"key\": \"value\"}",
			expected: `{"key": "value"}`,
		},
		{
			name:     "Backticks only at end",
			input:    "{\"key\": \"value\"}```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "Json prefix without backticks",
			input:    "json{\"key\": \"value\"}",
			expected: `{"key": "value"}`,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Only backticks",
			input:    "```",
			expected: "",
		},
		{
			name:     "Only json prefix",
			input:    "json",
			expected: "",
		},
		{
			name:     "Json with escaped characters",
			input:    "```json{\"key\": \"value\\n\"}```",
			expected: `{"key": "value\n"}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := fixLLmJson(test.input)
			if result != test.expected {
				t.Errorf("got %q, want %q", result, test.expected)
			}
		})
	}
}
