package options

import (
	"image/color"

	"github.com/vertexcover-io/locatr/golang/llm"
	"github.com/vertexcover-io/locatr/golang/types"
)

// LocatrOptions configures the behavior of the Locatr instance.
type LocatrOptions struct {
	// LLMClient handles interactions with the Language Learning Model
	LLMClient *llm.LLMClient
	// ReRanker provides document re-ranking capabilities
	ReRanker types.ReRankerInterface
	// CachePath specifies the file location for persisting locator cache
	CachePath string
	// UseCache enables caching of locator results
	UseCache bool
}

// HighlightOptions defines the options for highlighting a locator on a screenshot.
type HighlightOptions struct {
	// Color is the color to use for the highlight.
	Color *color.RGBA
	// Radius is the radius of the highlight.
	Radius int
	// Opacity is the opacity of the highlight.
	Opacity float64
}
