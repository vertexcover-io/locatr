package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"reflect"
	"strconv"
	"strings"

	"github.com/antchfx/xmlquery"
	"github.com/kaptinlin/jsonrepair"
	"github.com/vertexcover-io/locatr/pkg/types"
	"golang.org/x/net/html"
)

// ParseElementSpec parses the result of a minified DOM into an ElementSpec.
// Parameters:
//   - result: The result of a minified DOM
//
// Returns:
//   - *types.ElementSpec: The parsed element spec
//   - error: Any error that occurred during the parsing
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

// ParseLocatorMap parses the result of a locator map into a map of string slices.
// Parameters:
//   - result: The result of a locator map
//
// Returns:
//   - map[string][]string: The parsed locator map
//   - error: Any error that occurred during the parsing
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

// ParseLocators parses the result of a locators into a slice of strings.
// Parameters:
//   - result: The result of a locators
//
// Returns:
//   - []string: The parsed locators
//   - error: Any error that occurred during the parsing
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

// ParseLocation parses the result of a location into a Location.
// Parameters:
//   - result: The result of a location
//
// Returns:
//   - *types.Location: The parsed location
//   - error: Any error that occurred during the parsing
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
//   - results: Array of RerankResult containing relevance scores and indices
//
// Returns a new array containing only the valid chunks, ordered by their relevance scores.
// If no valid results are found, returns the original chunks array unchanged.
func SortRerankChunks(chunks []string, results []types.RerankResult) []string {
	// Filter out results with indices out of range
	validResults := []types.RerankResult{}
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

// ExtractFirstUniqueXMLID finds and returns the first ID attribute from a top-level element in an XML fragment.
// Parameters:
//   - xmlFragment: A string containing XML markup to analyze
//
// Returns:
//   - string: The first ID attribute value found
//   - error: If no ID is found or if XML parsing fails
func ExtractFirstUniqueXMLID(xmlFragment string) (string, error) {
	// Handle empty input
	if strings.TrimSpace(xmlFragment) == "" {
		return "", errors.New("empty XML fragment provided")
	}

	// Wrap the fragment in a root element for proper parsing
	wrappedXML := "<root>" + xmlFragment + "</root>"

	// Parse the XML using xmlquery
	doc, err := xmlquery.Parse(strings.NewReader(wrappedXML))
	if err != nil {
		return "", fmt.Errorf("error parsing XML: %w", err)
	}

	// Use XPath to find the first element with a non-empty id attribute
	// This query looks for any element with an id attribute that's not empty
	node := xmlquery.FindOne(doc, "//*[@id and @id!='']")
	if node != nil {
		return node.SelectAttr("id"), nil
	}

	return "", errors.New("no non-empty ID attribute found in the XML fragment")
}

// ExtractFirstUniqueHTMLID finds and returns the first ID attribute from a top-level element in an HTML fragment.
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
func ExtractFirstUniqueHTMLID(htmlFragment string) (string, error) {
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

// ParseFloatValue tries to parse the given value to a float64.
//
// Parameters:
//   - v: The value to parse
//
// Returns:
//   - float64: The parsed value
//   - error: An error if the given value is not a float64, int, or string
func ParseFloatValue(v any) (float64, error) {
	switch t := v.(type) {
	case float64:
		return t, nil
	case int:
		return float64(t), nil
	case string:
		return strconv.ParseFloat(t, 64)
	default:
		return 0, fmt.Errorf("unexpected type for float value: %T", v)
	}
}

// GetStructFields returns the struct name and field values from an interface.
// Parameters:
//   - i: The interface to get the fields from
//   - fieldNames: The names of the fields to retrieve
//
// Returns:
//   - string: The name of the struct
//   - map[string]reflect.Value: A map of field names to their reflected values
//   - error: An error if the interface is not a struct
func GetStructFields(i any, fieldNames ...string) (structName string, fields map[string]reflect.Value, err error) {
	v := reflect.ValueOf(i)

	// If it's a pointer, dereference it
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Ensure it's a struct
	if v.Kind() != reflect.Struct {
		return "", nil, fmt.Errorf("expected struct, got %s", v.Kind())
	}

	// Get struct name
	structName = v.Type().Name()
	fields = make(map[string]reflect.Value)

	// Retrieve requested fields
	for _, fieldName := range fieldNames {
		fields[fieldName] = v.FieldByName(fieldName)
	}

	return structName, fields, nil
}

// ParseJSON parses a JSON string from a text string and repairs it if possible.
// Parameters:
//   - text: The text to parse
//
// Returns:
//   - string: The parsed JSON
//   - error: Any error that occurred during the parsing
func ParseJSON(text string) (string, error) {
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimPrefix(text, "json")
	text = strings.TrimSuffix(text, "```")

	return jsonrepair.JSONRepair(text)
}

// ScaleAndPadImage scales an image while maintaining aspect ratio and adds off-white padding to fit the target resolution.
//
// Parameters:
//   - input: The input image in bytes
//   - targetResolution: The target resolution
//
// Returns:
//   - image.Image: The scaled and padded image
//   - error: Any error that occurred during the scaling and padding
func ScaleAndPadImage(img image.Image, targetResolution *types.Resolution) image.Image {
	origBounds := img.Bounds()
	origWidth := origBounds.Dx()
	origHeight := origBounds.Dy()

	// Compute the scale factor to fit the image into the target dimensions.
	scale := math.Min(
		float64(targetResolution.Width)/float64(origWidth),
		float64(targetResolution.Height)/float64(origHeight),
	)
	newWidth := int(math.Round(float64(origWidth) * scale))
	newHeight := int(math.Round(float64(origHeight) * scale))

	// Resize the original image.
	// Here we use a simple nearest neighbor approach; you might want a more sophisticated algorithm.
	resized := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			// Compute the corresponding pixel in the source image.
			srcX := int(math.Round(float64(x) / scale))
			srcY := int(math.Round(float64(y) / scale))
			// Clamp to the source bounds if necessary.
			if srcX >= origWidth {
				srcX = origWidth - 1
			}
			if srcY >= origHeight {
				srcY = origHeight - 1
			}
			resized.Set(x, y, img.At(srcX+origBounds.Min.X, srcY+origBounds.Min.Y))
		}
	}

	// Create a new image with the target dimensions, filling it with black.
	padded := image.NewRGBA(image.Rect(0, 0, targetResolution.Width, targetResolution.Height))
	draw.Draw(padded, padded.Bounds(), &image.Uniform{color.Black}, image.Point{}, draw.Src)

	// Calculate padding offsets to center the resized image.
	padX := (targetResolution.Width - newWidth) / 2
	padY := (targetResolution.Height - newHeight) / 2

	// Draw the resized image onto the padded image.
	offsetRect := image.Rect(padX, padY, padX+newWidth, padY+newHeight)
	draw.Draw(padded, offsetRect, resized, image.Point{}, draw.Over)

	return padded
}

// RemapPoint maps a point from the padded image coordinates back to the original image coordinates.
//
// Parameters:
//   - point: The point in the padded image (e.g., where the user clicked)
//   - originalResolution: The resolution of the original image
//   - targetResolution: The resolution of the target image
//
// Returns:
//   - *types.Point: The remapped point in the original image's coordinates
func RemapPoint(point *types.Point, originalResolution *types.Resolution, targetResolution *types.Resolution) *types.Point {
	// Compute the scale factor used in Letterbox.
	scale := math.Min(float64(targetResolution.Width)/float64(originalResolution.Width), float64(targetResolution.Height)/float64(originalResolution.Height))
	newWidth := int(math.Round(float64(originalResolution.Width) * scale))
	newHeight := int(math.Round(float64(originalResolution.Height) * scale))

	// Calculate the padding offsets.
	padX := (targetResolution.Width - newWidth) / 2
	padY := (targetResolution.Height - newHeight) / 2

	// Check if the coordinate falls within the region containing the scaled image.
	if point.X < float64(padX) || point.X >= float64(padX+newWidth) || point.Y < float64(padY) || point.Y >= float64(padY+newHeight) {
		// The coordinate is in the padded area.
		return nil
	}

	// Convert padded coordinate to scaled image coordinate.
	origX := int(math.Round((point.X - float64(padX)) / scale))
	origY := int(math.Round((point.Y - float64(padY)) / scale))

	// Clamp the result to the original image bounds.
	if origX >= originalResolution.Width {
		origX = originalResolution.Width - 1
	}
	if origY >= originalResolution.Height {
		origY = originalResolution.Height - 1
	}
	return &types.Point{X: float64(origX), Y: float64(origY)}
}

// RemapPointInverse maps a point from the original image coordinates to the padded image coordinates.
func RemapPointInverse(point *types.Point, originalResolution *types.Resolution, targetResolution *types.Resolution) *types.Point {
	// Compute the scale factor used in letterboxing.
	scale := math.Min(float64(targetResolution.Width)/float64(originalResolution.Width), float64(targetResolution.Height)/float64(originalResolution.Height))

	// Compute new dimensions of the scaled image.
	newWidth := int(math.Round(float64(originalResolution.Width) * scale))
	newHeight := int(math.Round(float64(originalResolution.Height) * scale))

	// Calculate padding offsets.
	padX := (targetResolution.Width - newWidth) / 2
	padY := (targetResolution.Height - newHeight) / 2

	// Map from original to padded coordinates.
	targetX := float64(padX) + (point.X * scale)
	targetY := float64(padY) + (point.Y * scale)

	return &types.Point{X: targetX, Y: targetY}
}

// DrawPoint draws a point on an image.
// Parameters:
//   - img: The image to draw on
//   - point: The point to draw
//   - opts: The options for the draw
func DrawPoint(img *image.RGBA, point *types.Point, config *types.HighlightConfig) {
	// Adjust the color's alpha based on the opacity
	alpha := uint8(config.Opacity * 255)
	highlightColor := color.RGBA{
		R: config.Color.R,
		G: config.Color.G,
		B: config.Color.B,
		A: alpha,
	}

	for dx := -config.Radius; dx <= config.Radius; dx++ {
		for dy := -config.Radius; dy <= config.Radius; dy++ {
			if dx*dx+dy*dy <= config.Radius*config.Radius { // Circle formula
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
