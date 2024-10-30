package locatr

import (
	"context"
	"errors"
	"time"

	"github.com/liushuangls/go-anthropic/v2"
	"github.com/sashabaranov/go-openai"
	"gopkg.in/validator.v2"
)

var ErrInvalidProviderForLlm = errors.New("invalid provider for llm")

type LlmProvider string

const MaxTokens int = 256

const (
	OpenAI    LlmProvider = "openai"
	Anthropic LlmProvider = "anthropic"
)

type llmClient struct {
	apiKey          string      `validate:"regexp=sk-*"`
	provider        LlmProvider `validate:"regexp=(openai|anthropic)"`
	model           string      `validate:"min=2,max=50"`
	openaiClient    *openai.Client
	anthropicClient *anthropic.Client
	completions     []chatCompletionResponse
}

type LlmClientInterface interface {
	ChatCompletion(prompt string) (*chatCompletionResponse, error)
	GetAllCompletionResponses() []chatCompletionResponse
}

type chatCompletionResponse struct {
	Prompt       string      `json:"prompt"`
	Completion   string      `json:"completion"`
	TotalTokens  int         `json:"total_tokens"`
	InputTokens  int         `json:"input_tokens"`
	OutputTokens int         `json:"output_tokens"`
	TimeTaken    int         `json:"time_taken"`
	Provider     LlmProvider `json:"provider"`
}

// NewLlmClient creates a new LLM client based on the specified provider, model, and API key.
// The `provider` parameter specifies the LLM provider (options: "openai" or "anthropic").
// The `model` parameter is the name of the model to use for the completion (e.g., "gpt-4o").
// The `apiKey` is the API key associated with the chosen provider.
// Returns an initialized *llmClient or an error if validation or provider initialization fails.
func NewLlmClient(provider LlmProvider, model string, apiKey string) (*llmClient, error) {
	client := &llmClient{
		apiKey:      apiKey,
		provider:    provider,
		model:       model,
		completions: []chatCompletionResponse{},
	}
	validate := validator.NewValidator()
	if err := validate.Validate(client); err != nil {
		return nil, err
	}

	switch client.provider {
	case OpenAI:
		client.openaiClient = openai.NewClient(client.apiKey)
	case Anthropic:
		client.anthropicClient = anthropic.NewClient(client.apiKey)
	default:
		return nil, ErrInvalidProviderForLlm
	}
	return client, nil
}

// ChatCompletion sends a prompt to the LLM model and returns the completion string or and error.
func (c *llmClient) ChatCompletion(prompt string) (*chatCompletionResponse, error) {
	switch c.provider {
	case OpenAI:
		return c.openaiRequest(prompt)
	case Anthropic:
		return c.anthropicRequest(prompt)
	default:
		return nil, ErrInvalidProviderForLlm
	}
}

func (c *llmClient) anthropicRequest(prompt string) (*chatCompletionResponse, error) {
	start := time.Now()
	resp, err := c.anthropicClient.CreateMessages(
		context.Background(),
		anthropic.MessagesRequest{
			Model: c.model,
			Messages: []anthropic.Message{
				anthropic.NewUserTextMessage(prompt),
			},
			MaxTokens: MaxTokens,
		})
	if err != nil {
		return nil, err
	}
	completionResponse := chatCompletionResponse{
		Prompt:       prompt,
		Completion:   resp.Content[0].GetText(),
		TotalTokens:  resp.Usage.OutputTokens + resp.Usage.InputTokens,
		InputTokens:  resp.Usage.InputTokens,
		OutputTokens: resp.Usage.OutputTokens,
		TimeTaken:    int(time.Since(start).Seconds()),
		Provider:     Anthropic,
	}
	c.completions = append(c.completions, completionResponse)

	return &completionResponse, nil
}

func (c *llmClient) openaiRequest(prompt string) (*chatCompletionResponse, error) {
	start := time.Now()
	resp, err := c.openaiClient.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: c.model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
		},
	)
	if err != nil {
		return nil, err
	}
	completionResponse := chatCompletionResponse{
		Prompt:       prompt,
		Completion:   resp.Choices[0].Message.Content,
		TotalTokens:  resp.Usage.TotalTokens,
		InputTokens:  resp.Usage.PromptTokens,
		OutputTokens: resp.Usage.CompletionTokens,
		TimeTaken:    int(time.Since(start).Seconds()),
		Provider:     OpenAI,
	}
	c.completions = append(c.completions, completionResponse)

	return &completionResponse, nil
}

func (c *llmClient) GetAllCompletionResponses() []chatCompletionResponse {
	return c.completions
}
