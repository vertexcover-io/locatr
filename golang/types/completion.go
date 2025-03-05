package types

// LLMCompletionMeta contains metadata about a language model completion.
type LLMCompletionMeta struct {
	TimeTaken    int         `json:"time_taken_by_llm"` // Time taken by the LLM to generate the completion
	InputTokens  int         `json:"input_tokens"`      // Number of input tokens used
	OutputTokens int         `json:"output_tokens"`     // Number of output tokens generated
	TotalTokens  int         `json:"total_tokens"`      // Total number of tokens (input + output)
	Provider     LLMProvider `json:"llm_provider"`      // Provider of the language model
	Model        string      `json:"llm_model"`         // Model used for the completion
}

// RawCompletion represents a raw completion response from a language model.
type RawCompletion struct {
	Text string // The text of the completion
	LLMCompletionMeta
}

// IdFromDOMCompletion represents the result of extracting an ID from a DOM.
type IdFromDOMCompletion struct {
	Id           string // The extracted ID
	ErrorMessage string // Error message if extraction failed
	LLMCompletionMeta
}

// PointFromScreenshotCompletion represents the result of extracting a point from a screenshot.
type PointFromScreenshotCompletion struct {
	Point        Point  // The extracted point
	ErrorMessage string // Error message if extraction failed
	LLMCompletionMeta
}

// LocatrCompletion represents the result of a locator completion.
type LocatrCompletion struct {
	Locators    []string    // List of locators found
	LocatorType locatorType // Type of locators in the list
	CacheHit    bool        // Indicates if the result was a cache hit
	LLMCompletionMeta
}
