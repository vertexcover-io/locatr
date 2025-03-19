package types

import (
	"image/color"
	"math"
)

// Ptr returns a pointer to a value.
func Ptr[T any](v T) *T {
	return &v
}

// Point represents a coordinate point in a 2D space.
type Point struct {
	X float64 `json:"x"` // X-coordinate
	Y float64 `json:"y"` // Y-coordinate
}

// Equals checks if two points are equal within a small tolerance.
func (p1 Point) Equals(p2 Point) bool {
	return math.Abs(p1.X-p2.X) < 1e-9 && math.Abs(p1.Y-p2.Y) < 1e-9
}

// Resolution represents the resolution of a viewport/window.
type Resolution struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// HighlightConfig defines the configuration for highlighting a locator.
type HighlightConfig struct {
	// Color is the color to use for the highlight. Defaults to red.
	Color *color.RGBA
	// Radius is the radius to use for the highlight. Defaults to 10.
	Radius int
	// Opacity is the opacity to use for the highlight. Defaults to 0.5.
	Opacity float64
	// Resolution is the resolution to use for the screenshot. Defaults to 1280x800.
	Resolution *Resolution
}
