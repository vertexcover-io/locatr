package locatr

import (
	"context"
	"errors"

	"github.com/liushuangls/go-anthropic/v2"
	"github.com/sashabaranov/go-openai"
	"gopkg.in/validator.v2"
)

var ErrInvalidProviderForLlm = errors.New("invalid provider for llm")

const MaxTokens int = 256

type llmClient struct {
	apiKey   string `validate:"regexp=sk-*"`
	provider string `validate:"regexp=(openai|anthropic)"`
	model    string `validate:"min=2,max=50"`

	openaiClient    *openai.Client
	anthropicClient *anthropic.Client
}

// NewLlmClient creates a new LLM client based on the specified provider, model, and API key.
// The `provider` parameter specifies the LLM provider (options: "openai" or "anthropic").
// The `model` parameter is the name of the model to use for the completion (e.g., "gpt-4o").
// The `apiKey` is the API key associated with the chosen provider.
// Returns an initialized *llmClient or an error if validation or provider initialization fails.
func NewLlmClient(provider string, model string, apiKey string) (*llmClient, error) {
	client := &llmClient{
		apiKey:   apiKey,
		provider: provider,
		model:    model,
	}
	validate := validator.NewValidator()
	if err := validate.Validate(client); err != nil {
		return nil, err
	}

	switch client.provider {
	case "openai":
		client.openaiClient = openai.NewClient(client.apiKey)
	case "anthropic":
		client.anthropicClient = anthropic.NewClient(client.apiKey)
	default:
		return nil, ErrInvalidProviderForLlm
	}

	return client, nil
}

// ChatCompletion sends a prompt to the LLM model and returns the completion string or and error.
func (c *llmClient) ChatCompletion(prompt string) (string, error) {
	switch c.provider {
	case "openai":
		return c.openaiRequest(prompt)
	case "anthropic":
		return c.anthropicRequest(prompt)
	default:
		return "", ErrInvalidProviderForLlm
	}
}

func (c *llmClient) anthropicRequest(prompt string) (string, error) {
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
		return "", err
	}

	return resp.Content[0].GetText(), nil
}

func (c *llmClient) openaiRequest(prompt string) (string, error) {
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
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}
