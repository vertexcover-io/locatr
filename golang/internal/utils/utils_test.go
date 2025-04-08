package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vertexcover-io/locatr/golang/types"
)

func TestParseElementSpec(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		want    *types.ElementSpec
		wantErr bool
	}{
		{
			name: "Valid JSON string",
			input: `{
				"id": "test-id",
				"tag_name": "div",
				"text": "test",
				"attributes": {"class": "test-class"},
				"children": []
			}`,
			want: &types.ElementSpec{
				Id:         "test-id",
				TagName:    "div",
				Text:       "test",
				Attributes: map[string]string{"class": "test-class"},
				Children:   []types.ElementSpec{},
			},
		},
		{
			name:    "Invalid JSON string",
			input:   `{"id":}`,
			wantErr: true,
		},
		{
			name:    "Non-string input",
			input:   123,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseElementSpec(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseLocatorMap(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		want    map[string][]string
		wantErr bool
	}{
		{
			name:  "Valid JSON string",
			input: `{"css":["#id",".class"],"xpath":["//*[@id='test']"]}`,
			want:  map[string][]string{"css": {"#id", ".class"}, "xpath": {"//*[@id='test']"}},
		},
		{
			name:    "Invalid JSON string",
			input:   `{"css":}`,
			wantErr: true,
		},
		{
			name:    "Non-string input",
			input:   123,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseLocatorMap(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseLocators(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		want    []string
		wantErr bool
	}{
		{
			name:  "Valid JSON string",
			input: `["#id",".class"]`,
			want:  []string{"#id", ".class"},
		},
		{
			name:    "Invalid JSON string",
			input:   `["}`,
			wantErr: true,
		},
		{
			name:    "Non-string input",
			input:   123,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseLocators(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseLocation(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		want    *types.Location
		wantErr bool
	}{
		{
			name: "Valid JSON string",
			input: `{
				"point": {"x": 100, "y": 200},
				"scroll_position": {"x": 0, "y": 500}
			}`,
			want: &types.Location{
				Point:          types.Point{X: 100, Y: 200},
				ScrollPosition: types.Point{X: 0, Y: 500},
			},
		},
		{
			name:    "Invalid JSON string",
			input:   `{"point":}`,
			wantErr: true,
		},
		{
			name:    "Non-string input",
			input:   123,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseLocation(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseLocatorValidationResult(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		want    bool
		wantErr bool
	}{
		{
			name:  "Boolean true",
			input: true,
			want:  true,
		},
		{
			name:  "Boolean false",
			input: false,
			want:  false,
		},
		{
			name:  "String true",
			input: "true",
			want:  true,
		},
		{
			name:  "String false",
			input: "false",
			want:  false,
		},
		{
			name:    "Invalid string",
			input:   "invalid",
			wantErr: true,
		},
		{
			name:    "Invalid type",
			input:   123,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseLocatorValidationResult(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSortRerankChunks(t *testing.T) {
	tests := []struct {
		name    string
		chunks  []string
		results []types.RerankResult
		want    []string
	}{
		{
			name:   "Valid rerank results",
			chunks: []string{"first", "second", "third"},
			results: []types.RerankResult{
				{Index: 2, Score: 0.9},
				{Index: 0, Score: 0.8},
				{Index: 1, Score: 0.7},
			},
			want: []string{"third", "first", "second"},
		},
		{
			name:   "Out of range indices",
			chunks: []string{"first", "second"},
			results: []types.RerankResult{
				{Index: 3, Score: 0.9},
				{Index: 4, Score: 0.8},
			},
			want: []string{"first", "second"},
		},
		{
			name:    "Empty results",
			chunks:  []string{"first", "second"},
			results: []types.RerankResult{},
			want:    []string{"first", "second"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SortRerankChunks(tt.chunks, tt.results)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractFirstUniqueID(t *testing.T) {
	tests := []struct {
		name    string
		html    string
		want    string
		wantErr bool
	}{
		{
			name: "Valid ID",
			html: `<div id="test">content</div>`,
			want: "test",
		},
		{
			name: "Multiple IDs",
			html: `<div id="first"><span id="second">content</span></div>`,
			want: "first",
		},
		{
			name:    "No ID",
			html:    `<div>content</div>`,
			wantErr: true,
		},
		{
			name:    "Invalid HTML",
			html:    `<div`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractFirstUniqueHTMLID(tt.html)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseFloatValue(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  float64
	}{
		{
			name:  "Float64 input",
			input: float64(123.45),
			want:  123.45,
		},
		{
			name:  "Integer input",
			input: 123,
			want:  123.0,
		},
		{
			name:  "String input",
			input: "123.45",
			want:  123.45,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFloatValue(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "Valid JSON",
			input: `{"key": "value"}`,
			want:  `{"key": "value"}`,
		},
		{
			name:  "JSON with code blocks",
			input: "```json\n{\"key\": \"value\"}\n```",
			want:  `{"key": "value"}`,
		},
		{
			name:    "Invalid JSON",
			input:   `"key":"value"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseJSON(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, strings.TrimSpace(got))
		})
	}
}
