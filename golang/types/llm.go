package types

// LLMProvider is a string alias representing a language model provider.
type LLMProvider = string

// LLMClientInterface defines the interface for a language model client required by locatr instance.
type LLMClientInterface interface {

	// GetProvider returns the provider of the LLM.
	GetProvider() LLMProvider

	// GetModel returns the model of the LLM.
	GetModel() string

	// GetJSONCompletion returns the JSON completion for the given prompt.
	GetJSONCompletion(prompt string, image []byte) (*JSONCompletion, error)
}

// LLMCompletionMeta contains metadata about a language model completion.
type LLMCompletionMeta struct {
	InputTokens  int         `json:"input_tokens"`  // Number of input tokens used
	OutputTokens int         `json:"output_tokens"` // Number of output tokens generated
	Provider     LLMProvider `json:"llm_provider"`  // Provider of the language model
	Model        string      `json:"llm_model"`     // Model used for the completion
}

// CalculateCost calculates the cost of the completion.
// Parameters:
//   - costPer1MInputTokens: Cost per 1 million input tokens
//   - costPer1MOutputTokens: Cost per 1 million output tokens
//
// Returns:
//   - float64: Total cost of the completion
func (c LLMCompletionMeta) CalculateCost(costPer1MInputTokens, costPer1MOutputTokens float64) float64 {
	inputCost := (float64(c.InputTokens) / 1000000.0) * costPer1MInputTokens
	outputCost := (float64(c.OutputTokens) / 1000000.0) * costPer1MOutputTokens
	return inputCost + outputCost
}

// JSONCompletion represents a JSON completion response from a language model.
type JSONCompletion struct {
	JSON string `json:"json_value"` // The JSON content of the completion
	LLMCompletionMeta
}
