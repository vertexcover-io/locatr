package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/vertexcover-io/locatr/golang/options"
	"github.com/vertexcover-io/locatr/golang/types"
	"golang.org/x/net/html"
)

func ParseElementSpec(result any) (*types.ElementSpec, error) {
	if resultStr, ok := result.(string); ok {
		elementSpec := &types.ElementSpec{}
		if err := json.Unmarshal([]byte(resultStr), &elementSpec); err != nil {
			return nil, fmt.Errorf("failed to unmarshal minified root element json: %v", err)
		}
		return elementSpec, nil
	}
	return nil, fmt.Errorf("unexpected type for minified DOM result: %T", result)
}

func ParseLocatorMap(result any) (map[string][]string, error) {
	if resultStr, ok := result.(string); ok {
		locatorMap := map[string][]string{}
		if err := json.Unmarshal([]byte(resultStr), &locatorMap); err != nil {
			return nil, fmt.Errorf("failed to read locator map: %v", err)
		}
		return locatorMap, nil
	}
	return nil, fmt.Errorf("unexpected type for locator map result: %T", result)
}

func ParseLocators(result any) ([]string, error) {
	if resultStr, ok := result.(string); ok {
		var locators []string
		if err := json.Unmarshal([]byte(resultStr), &locators); err != nil {
			return nil, fmt.Errorf("failed to read locators: %v", err)
		}
		return locators, nil
	}
	return nil, fmt.Errorf("unexpected type for locators result: %T", result)
}

func ParseLocation(result any) (*types.Location, error) {
	if resultStr, ok := result.(string); ok {
		location := &types.Location{}
		decoder := json.NewDecoder(strings.NewReader(resultStr))
		if err := decoder.Decode(&location); err != nil {
			return nil, fmt.Errorf("failed to read location: %v", err)
		}
		return location, nil
	}
	return nil, fmt.Errorf("unexpected type for location result: %T", result)
}

func ParseLocatorValidationResult(result any) (bool, error) {
	switch v := result.(type) {
	case bool:
		return v, nil
	case string:
		var isValid bool
		if err := json.Unmarshal([]byte(v), &isValid); err != nil {
			return false, fmt.Errorf("failed to parse locator validation result: %v", err)
		}
		return isValid, nil
	default:
		return false, fmt.Errorf("unexpected type for locator validation result: %T", result)
	}
}

// SortRerankChunks reorders a list of text chunks based on their relevance scores.
// Parameters:
//   - chunks: Original array of text chunks to be sorted
//   - results: Array of ReRankResult containing relevance scores and indices
//
// Returns a new array containing only the valid chunks, ordered by their relevance scores.
// If no valid results are found, returns the original chunks array unchanged.
func SortRerankChunks(chunks []string, results []types.ReRankResult) []string {
	// Filter out results with indices out of range
	validResults := []types.ReRankResult{}
	for _, result := range results {
		if result.Index < len(chunks) {
			validResults = append(validResults, result)
		}
	}

	// If no valid results, return the original chunks
	if len(validResults) == 0 {
		return chunks
	}

	// Sort chunks based on valid rerank results
	finalChunks := []string{}
	for _, result := range validResults {
		finalChunks = append(finalChunks, chunks[result.Index])
	}
	return finalChunks
}

// ExtractFirstUniqueID finds and returns the first ID attribute from a top-level element in an HTML fragment.
// Parameters:
//   - htmlFragment: A string containing HTML markup to analyze
//
// Returns:
//   - string: The first ID attribute value found
//   - error: If no ID is found or if HTML parsing fails
//
// The function works by:
// 1. Wrapping the fragment in a root element for proper parsing
// 2. Creating a DOM tree from the HTML
// 3. Traversing the tree to find the first element with an ID attribute
func ExtractFirstUniqueID(htmlFragment string) (string, error) {
	wrappedHTML := "<root>" + htmlFragment + "</root>"
	doc, err := html.Parse(strings.NewReader(wrappedHTML))
	if err != nil {
		return "", fmt.Errorf("error parsing HTML: %w", err)
	}

	// Find the artificial root node
	var rootNode *html.Node
	var findRoot func(*html.Node) bool
	findRoot = func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == "root" {
			rootNode = n
			return true
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if findRoot(c) {
				return true
			}
		}

		return false
	}

	findRoot(doc)

	// Look for the first element with an ID
	var firstID string
	var findFirstID func(*html.Node) bool
	findFirstID = func(n *html.Node) bool {
		if n.Type == html.ElementNode {
			for _, attr := range n.Attr {
				if attr.Key == "id" {
					firstID = attr.Val
					return true
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if findFirstID(c) {
				return true
			}
		}

		return false
	}

	if rootNode != nil {
		findFirstID(rootNode)
	}

	if firstID == "" {
		return "", errors.New("no ID attribute found in the HTML fragment")
	}

	return firstID, nil
}

// GetFloatValue converts a value to a float64.
// Parameters:
//   - v: The value to convert
//
// Returns:
//   - float64: The converted value
func GetFloatValue(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	default:
		return float64(v.(int))
	}
}

func HighlightPoint(point *types.Point, img *image.RGBA, opts *options.HighlightOptions) {
	// Adjust the color's alpha based on the opacity
	alpha := uint8(opts.Opacity * 255)
	highlightColor := color.RGBA{
		R: opts.Color.R,
		G: opts.Color.G,
		B: opts.Color.B,
		A: alpha,
	}

	for dx := -opts.Radius; dx <= opts.Radius; dx++ {
		for dy := -opts.Radius; dy <= opts.Radius; dy++ {
			if dx*dx+dy*dy <= opts.Radius*opts.Radius { // Circle formula
				X := int(point.X) + dx
				Y := int(point.Y) + dy
				if image.Pt(X, Y).In(img.Bounds()) {
					// Blend the highlight color with the existing pixel color
					originalColor := img.At(X, Y).(color.RGBA)
					blendedColor := blendColors(originalColor, highlightColor)
					img.Set(X, Y, blendedColor)
				}
			}
		}
	}
}

// blendColors blends two colors based on the alpha of the overlay color.
func blendColors(base, overlay color.RGBA) color.RGBA {
	alpha := float64(overlay.A) / 255.0
	invAlpha := 1.0 - alpha

	return color.RGBA{
		R: uint8(float64(base.R)*invAlpha + float64(overlay.R)*alpha),
		G: uint8(float64(base.G)*invAlpha + float64(overlay.G)*alpha),
		B: uint8(float64(base.B)*invAlpha + float64(overlay.B)*alpha),
		A: base.A, // Keep the base alpha
	}
}
