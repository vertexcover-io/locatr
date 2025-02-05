package llm

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/liushuangls/go-anthropic/v2"
	"github.com/openai/openai-go"
	openai_option "github.com/openai/openai-go/option"
	"github.com/vertexcover-io/locatr/golang/logger"
	"gopkg.in/validator.v2"
)

type LlmProvider string

type LlmWebInputDto struct {
	HtmlDom string `json:"html_dom"`
	UserReq string `json:"user_req"`
}

type llmClient struct {
	apiKey          string      `validate:"required"`
	provider        LlmProvider `validate:"regexp=(openai|anthropic|open-router|groq)"`
	model           string      `validate:"min=2,max=50"`
	openaiClient    *openai.Client
	anthropicClient *anthropic.Client
}

type LlmClientInterface interface {
	ChatCompletion(prompt string) (*ChatCompletionResponse, error)
	GetProvider() LlmProvider
	GetModel() string
}

type ChatCompletionResponse struct {
	Prompt       string      `json:"prompt"`
	Completion   string      `json:"completion"`
	TotalTokens  int         `json:"total_tokens"`
	InputTokens  int         `json:"input_tokens"`
	OutputTokens int         `json:"output_tokens"`
	TimeTaken    int         `json:"time_taken"`
	Provider     LlmProvider `json:"provider"`
}

const (
	OpenAI     LlmProvider = "openai"
	Anthropic  LlmProvider = "anthropic"
	OpenRouter LlmProvider = "open-router"
	Groq       LlmProvider = "groq"
)

var ErrInvalidProviderForLlm = errors.New("invalid provider for llm")

const MAX_TOKENS int = 256

// NewLlmClient creates a new LLM client based on the specified provider, model, and API key.
// The `provider` parameter specifies the LLM provider (options: "openai" or "anthropic").
// The `model` parameter is the name of the model to use for the completion (e.g., "gpt-4o").
// The `apiKey` is the API key associated with the chosen provider.
// Returns an initialized *llmClient or an error if validation or provider initialization fails.
func NewLlmClient(provider LlmProvider, model string, apiKey string) (*llmClient, error) {
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
	case OpenAI:
		os.Setenv("OPENAI_API_KEY", client.apiKey)
		client.openaiClient = openai.NewClient()
	case Anthropic:
		client.anthropicClient = anthropic.NewClient(client.apiKey)
	case OpenRouter:
		os.Setenv("OPENAI_API_KEY", client.apiKey)
		client.openaiClient = openai.NewClient(openai_option.WithBaseURL("https://openrouter.ai/api/v1/"))
	case Groq:
		os.Setenv("OPENAI_API_KEY", client.apiKey)
		client.openaiClient = openai.NewClient(openai_option.WithBaseURL("https://api.groq.com/openai/v1/"))
	default:
		return nil, ErrInvalidProviderForLlm
	}
	return client, nil
}

// ChatCompletion sends a prompt to the LLM model and returns the completion string or and error.
func (c *llmClient) ChatCompletion(prompt string) (*ChatCompletionResponse, error) {
	switch c.provider {
	case OpenAI, OpenRouter, Groq:
		return c.openaiRequest(prompt)
	case Anthropic:
		return c.anthropicRequest(prompt)
	default:
		return nil, ErrInvalidProviderForLlm
	}
}

func (c *llmClient) anthropicRequest(prompt string) (*ChatCompletionResponse, error) {
	defer logger.GetTimeLogger("LLM: AnthropicCompletion")()

	start := time.Now()
	resp, err := c.anthropicClient.CreateMessages(
		context.Background(),
		anthropic.MessagesRequest{
			Model: c.model,
			Messages: []anthropic.Message{
				anthropic.NewUserTextMessage(prompt),
			},
			MaxTokens: MAX_TOKENS,
		})
	if err != nil {
		return nil, err
	}
	completionResponse := ChatCompletionResponse{
		Prompt:       prompt,
		Completion:   resp.Content[0].GetText(),
		TotalTokens:  resp.Usage.OutputTokens + resp.Usage.InputTokens,
		InputTokens:  resp.Usage.InputTokens,
		OutputTokens: resp.Usage.OutputTokens,
		TimeTaken:    int(time.Since(start).Seconds()),
		Provider:     Anthropic,
	}

	return &completionResponse, nil
}

func (c *llmClient) openaiRequest(prompt string) (*ChatCompletionResponse, error) {
	defer logger.GetTimeLogger("LLM: OpenAICompletion")()

	start := time.Now()
	resp, err := c.openaiClient.Chat.Completions.New(
		context.Background(), openai.ChatCompletionNewParams{
			Model: openai.F(c.model),
			Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
				openai.UserMessage(prompt),
			}),
			ResponseFormat: openai.F[openai.ChatCompletionNewParamsResponseFormatUnion](
				openai.ResponseFormatJSONObjectParam{
					Type: openai.F(openai.ResponseFormatJSONObjectTypeJSONObject),
				},
			),
		},
	)
	if err != nil {
		return nil, err
	}
	completionResponse := ChatCompletionResponse{
		Prompt:       prompt,
		Completion:   resp.Choices[0].Message.Content,
		TotalTokens:  int(resp.Usage.TotalTokens),
		InputTokens:  int(resp.Usage.PromptTokens),
		OutputTokens: int(resp.Usage.CompletionTokens),
		TimeTaken:    int(time.Since(start).Seconds()),
		Provider:     OpenAI,
	}

	return &completionResponse, nil
}

func CreateLlmClientFromEnv() (*llmClient, error) {
	return NewLlmClient(
		LlmProvider(os.Getenv("LLM_PROVIDER")), os.Getenv("LLM_MODEL"), os.Getenv("LLM_API_KEY"),
	)
}

func (c *llmClient) GetProvider() LlmProvider {
	return c.provider
}

func (c *llmClient) GetModel() string {
	return c.model
}
