package llm

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/liushuangls/go-anthropic/v2"
	"github.com/sashabaranov/go-openai"
	"gopkg.in/validator.v2"
)

var (
	ErrInvalidProviderForLlm = errors.New("invalid provider for LLM")
	MaxTokens                = 256
)

type HttpPostFunc = func(url string, headers map[string]string, body []byte) ([]byte, error)

type llmClient struct {
	apiKey   string `validate:"regexp=sk-*"`
	provider string `validate:"regexp=(openai|anthropic)"`
	model    string `validate:"min=2,max=50"`

	openaiClient    *openai.Client
	anthropicClient *anthropic.Client

	httpPost HttpPostFunc
}

func NewLlmClient(provider, model, apiKey string, postFunc HttpPostFunc) (*llmClient, error) {
	client := &llmClient{
		apiKey:   apiKey,
		provider: provider,
		model:    model,
		httpPost: postFunc,
	}

	if err := validator.Validate(client); err != nil {
		return nil, err
	}

	switch provider {
	case "openai":
		client.openaiClient = openai.NewClient(apiKey)
	case "anthropic":
		client.anthropicClient = anthropic.NewClient(apiKey)
	default:
		return nil, ErrInvalidProviderForLlm
	}

	fmt.Println("LLM Client created with provider:", provider)
	return client, nil
}

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
	request := anthropic.MessagesRequest{
		Model: c.model,
		Messages: []anthropic.Message{
			anthropic.NewUserTextMessage(prompt),
		},
		MaxTokens: MaxTokens,
	}

	payload, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal Anthropic request: %w", err)
	}

	url := "https://api.anthropic.com/v1/complete"
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": fmt.Sprintf("Bearer %s", c.apiKey),
	}

	response, err := c.httpPost(url, headers, payload)
	if err != nil {
		return "", fmt.Errorf("failed to send Anthropic request: %w", err)
	}

	var resp anthropic.MessagesResponse
	if err := json.Unmarshal(response, &resp); err != nil {
		return "", fmt.Errorf("failed to unmarshal Anthropic response: %w", err)
	}

	return resp.Content[0].GetText(), nil
}

func (c *llmClient) openaiRequest(prompt string) (string, error) {
	request := openai.ChatCompletionRequest{
		Model: c.model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "You are a helpful assistant.",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
	}

	payload, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal OpenAI request: %w", err)
	}

	url := "https://api.openai.com/v1/chat/completions"
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": fmt.Sprintf("Bearer %s", c.apiKey),
	}

	response, err := c.httpPost(url, headers, payload)
	if err != nil {
		return "", fmt.Errorf("failed to send OpenAI request: %w", err)
	}

	var resp openai.ChatCompletionResponse
	if err := json.Unmarshal(response, &resp); err != nil {
		return "", fmt.Errorf("failed to unmarshal OpenAI response: %w", err)
	}

	return resp.Choices[0].Message.Content, nil
}

