package reranker

import (
	"reflect"
	"testing"
)

func TestSplitWithSeparator(t *testing.T) {
	testCases := []struct {
		name      string
		text      string
		separator string
		expected  []string
	}{
		{
			name:      "Basic separator",
			text:      "hello,world,test",
			separator: ",",
			expected:  []string{"hello", ",", "world", ",", "test"},
		},
		{
			name:      "No separator in text",
			text:      "helloworld",
			separator: ",",
			expected:  []string{"helloworld"},
		},
		{
			name:      "Empty text",
			text:      "",
			separator: ",",
			expected:  []string{""},
		},
		{
			name:      "Separator at beginning and end",
			text:      ",hello,world,",
			separator: ",",
			expected:  []string{"", ",", "hello", ",", "world", ",", ""},
		},
		{
			name:      "Complex regex separator",
			text:      "hello123world456test",
			separator: "\\d+",
			expected:  []string{"hello", "123", "world", "456", "test"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := splitWithSeparator(tc.text, tc.separator)
			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestSplitTextWithRegex(t *testing.T) {
	testCases := []struct {
		name          string
		text          string
		separator     string
		keepSeparator bool
		expected      []string
	}{
		{
			name:          "Keep separator",
			text:          "hello,world,test",
			separator:     ",",
			keepSeparator: true,
			expected:      []string{"hello", ",world", ",test"},
		},
		{
			name:          "Discard separator",
			text:          "hello,world,test",
			separator:     ",",
			keepSeparator: false,
			expected:      []string{"hello", ",", "world", ",", "test"},
		},
		{
			name:          "Empty text",
			text:          "",
			separator:     ",",
			keepSeparator: false,
			expected:      []string{},
		},
		{
			name:          "No separator",
			text:          "helloworld",
			separator:     "",
			keepSeparator: false,
			expected:      []string{"helloworld"},
		},
		{
			name:          "Complex regex separator with keep",
			text:          "hello123world456test",
			separator:     "\\d+",
			keepSeparator: true,
			expected:      []string{"hello", "123world", "456test"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := splitTextWithRegex(tc.text, tc.separator, tc.keepSeparator)
			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestJoinDocs(t *testing.T) {
	testCases := []struct {
		name            string
		docs            []string
		separator       string
		stripWhitespace bool
		expected        *string
	}{
		{
			name:            "Basic join",
			docs:            []string{"hello", "world"},
			separator:       " ",
			stripWhitespace: false,
			expected:        stringPtr("hello world"),
		},
		{
			name:            "Strip whitespace",
			docs:            []string{"\thello", "world\t"},
			separator:       ", ",
			stripWhitespace: true,
			expected:        stringPtr("hello, world"),
		},
		{
			name:            "Empty docs",
			docs:            []string{},
			separator:       " ",
			stripWhitespace: false,
			expected:        nil,
		},
		{
			name:            "Single doc",
			docs:            []string{"hello"},
			separator:       " ",
			stripWhitespace: false,
			expected:        stringPtr("hello"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := joinDocs(tc.docs, tc.separator, tc.stripWhitespace)
			if (result == nil) != (tc.expected == nil) {
				t.Errorf("Expected nil status to be %v, got %v", tc.expected == nil, result == nil)
			}
			if result != nil && tc.expected != nil && *result != *tc.expected {
				t.Errorf("Expected %v, got %v", *tc.expected, *result)
			}
		})
	}
}

func TestMergeSplits(t *testing.T) {
	testCases := []struct {
		name         string
		splits       []string
		separator    string
		maxChunkSize int
		expected     []string
	}{
		{
			name:         "Chunk size limitation",
			splits:       []string{"hello", "verylongwordthatexceedschunksize", "test"},
			separator:    " ",
			maxChunkSize: 10,
			expected:     []string{"hello", "verylongwordthatexceedschunksize", "test"},
		},
		{
			name:         "Empty splits",
			splits:       []string{},
			separator:    " ",
			maxChunkSize: 100,
			expected:     []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := mergeSplits(tc.splits, tc.separator, tc.maxChunkSize)
			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestSplitHtml(t *testing.T) {
	testCases := []struct {
		name       string
		text       string
		separators []string
		chunkSize  int
		expected   []string
	}{
		{
			name:       "Basic HTML splitting",
			text:       "<div>hello</div><p>world</p>",
			separators: []string{"<[^>]+>", "</.+>", ""},
			chunkSize:  20,
			expected:   []string{"<div>hello</div>", "</div><p>world</p>"},
		},
		{
			name:       "No separators match",
			text:       "plain text content",
			separators: []string{"<[^>]+>", "</.+>"},
			chunkSize:  10,
			expected:   []string{"plain text content"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := SplitHtml(tc.text, tc.separators, tc.chunkSize)
			if !reflect.DeepEqual(result, tc.expected) {

				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
