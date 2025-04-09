package llm

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	anthropicOption "github.com/anthropics/anthropic-sdk-go/option"
	"github.com/openai/openai-go"
	openaiOption "github.com/openai/openai-go/option"
	"github.com/vertexcover-io/locatr/pkg/internal/utils"
	"github.com/vertexcover-io/locatr/pkg/logging"
	"github.com/vertexcover-io/locatr/pkg/types"
)

// LLMProvider constants define the supported Language Model service providers
const (
	OpenAI     types.LLMProvider = "openai"      // OpenAI API provider (e.g., GPT-4, GPT-3.5)
	Anthropic  types.LLMProvider = "anthropic"   // Anthropic API provider (e.g., Claude models)
	Groq       types.LLMProvider = "groq"        // Groq API provider
	OpenRouter types.LLMProvider = "open-router" // OpenRouter API aggregation service
)

type config struct {
	provider types.LLMProvider
	model    string
	apiKey   string
	logger   *slog.Logger
}

type Option func(*config)

// WithProvider sets the LLM provider for the configuration.
func WithProvider(provider types.LLMProvider) Option {
	return func(c *config) {
		c.provider = provider
	}
}

// WithModel sets the LLM model for the configuration.
func WithModel(model string) Option {
	return func(c *config) {
		c.model = model
	}
}

// WithAPIKey sets the API key for the configuration.
func WithAPIKey(apiKey string) Option {
	return func(c *config) {
		c.apiKey = apiKey
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(c *config) {
		c.logger = logger
	}
}

// llmClient represents a client for interacting with Language Model APIs.
// It encapsulates the provider configuration and json completion request handler.
type llmClient struct {
	config  *config
	handler func(ctx context.Context, prompt string, image []byte) (*types.JSONCompletion, error)
}

// NewLLMClient creates a new LLM client instance with the specified configuration.
// Parameters:
//   - opts: Configuration options for the LLM client
//
// Returns:
//   - *llmClient: Configured client instance
//   - error: Any initialization errors
func NewLLMClient(opts ...Option) (*llmClient, error) {

	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.provider == "" {
		return nil, errors.New("llm provider is required")
	}
	if cfg.model == "" {
		return nil, errors.New("llm model Id is required")
	}
	if cfg.logger == nil {
		cfg.logger = logging.DefaultLogger
	}

	var handler func(ctx context.Context, prompt string, image []byte) (*types.JSONCompletion, error)
	switch cfg.provider {
	case Anthropic:
		betas := []anthropic.AnthropicBeta{}
		if strings.HasPrefix(cfg.model, "claude-3-5-sonnet") {
			betas = append(betas, anthropic.AnthropicBetaComputerUse2024_10_22)
		}
		if strings.HasPrefix(cfg.model, "claude-3-7-sonnet") {
			betas = append(betas, anthropic.AnthropicBetaComputerUse2025_01_24)
		}
		handler = func(ctx context.Context, prompt string, image []byte) (*types.JSONCompletion, error) {
			client := anthropic.NewClient(anthropicOption.WithAPIKey(cfg.apiKey))
			return requestAnthropic(
				ctx, client, cfg.provider, cfg.model, prompt, image, &betas,
			)
		}

	case OpenAI, Groq, OpenRouter:
		options := []openaiOption.RequestOption{openaiOption.WithAPIKey(cfg.apiKey)}
		if cfg.provider == Groq {
			options = append(options, openaiOption.WithBaseURL("https://api.groq.com/openai/v1/"))
		} else if cfg.provider == OpenRouter {
			options = append(options, openaiOption.WithBaseURL("https://openrouter.ai/api/v1/"))
		}
		handler = func(ctx context.Context, prompt string, image []byte) (*types.JSONCompletion, error) {
			return requestOpenAI(
				ctx, openai.NewClient(options...), cfg.provider, cfg.model, prompt, image,
			)
		}

	default:
		return nil, errors.New("invalid provider for llm")
	}

	return &llmClient{config: cfg, handler: handler}, nil
}

var errDefaultLLMAPIKeyNotSet = errors.New("'LOCATR_ANTHROPIC_API_KEY' or 'ANTHROPIC_API_KEY' environment variable is not set")

// DefaultLLMClient returns a default LLM client using Anthropic's Claude 3.5 Sonnet model.
//
// Parameters:
//   - logger: The logger to use for logging
//
// Returns:
//   - *llmClient: Configured client instance
//   - error: Any initialization errors
func DefaultLLMClient(logger *slog.Logger) (*llmClient, error) {
	apiKey := os.Getenv("LOCATR_ANTHROPIC_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			return nil, errDefaultLLMAPIKeyNotSet
		}
	}
	options := []Option{
		WithProvider(Anthropic),
		WithModel("claude-3-5-sonnet-latest"),
		WithAPIKey(apiKey),
	}
	if logger != nil {
		options = append(options, WithLogger(logger))
	}
	return NewLLMClient(options...)
}

// GetJSONCompletion returns the JSON completion for the given prompt.
func (client *llmClient) GetJSONCompletion(ctx context.Context, prompt string, image []byte) (*types.JSONCompletion, error) {
	topic := fmt.Sprintf(
		"[LLM Completion] provider: %v, model: %v", client.config.provider, client.config.model,
	)
	defer logging.CreateTopic(topic, client.config.logger)()
	return client.handler(ctx, prompt, image)
}

// GetProvider returns the configured LLM service provider for this client.
func (client *llmClient) GetProvider() types.LLMProvider {
	return client.config.provider
}

// GetModel returns the configured model name for this client.
func (client *llmClient) GetModel() string {
	return client.config.model
}

// requestOpenAI handles API requests to OpenAI-compatible endpoints (OpenAI, Groq, OpenRouter).
//
// Parameters:
//   - ctx: Context
//   - client: Configured OpenAI API client
//   - provider: The service provider being used
//   - model: Model identifier
//   - prompt: The input prompt
//   - image: Optional image data for vision models
//
// Returns:
//   - *types.JSONCompletion: The API response and metadata
//   - error: Any API or processing errors
func requestOpenAI(
	ctx context.Context, client *openai.Client, provider types.LLMProvider, model, prompt string, image []byte,
) (*types.JSONCompletion, error) {
	completion := &types.JSONCompletion{
		LLMCompletionMeta: types.LLMCompletionMeta{
			InputTokens:  0,
			OutputTokens: 0,
			Provider:     provider,
			Model:        model,
		},
	}
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.UserMessage(prompt),
	}
	if image != nil {
		messages = append(messages, openai.UserMessageParts(
			openai.ImagePart(fmt.Sprintf(
				"data:image/jpeg;base64,%s", base64.StdEncoding.EncodeToString(image),
			)),
		))
	}

	response, err := client.Chat.Completions.New(
		ctx, openai.ChatCompletionNewParams{
			Model:    openai.F(model),
			Messages: openai.F(messages),
			ResponseFormat: openai.F[openai.ChatCompletionNewParamsResponseFormatUnion](
				openai.ResponseFormatJSONObjectParam{
					Type: openai.F(openai.ResponseFormatJSONObjectTypeJSONObject),
				},
			),
		},
	)
	if err != nil {
		return completion, fmt.Errorf("failed to get completion from %v: %w", provider, err)
	}

	completion.InputTokens = int(response.Usage.PromptTokens)
	completion.OutputTokens = int(response.Usage.CompletionTokens)

	jsonStr, err := utils.ParseJSON(response.Choices[0].Message.Content)
	if err != nil {
		return completion, fmt.Errorf("failed to parse JSON: %w", err)
	}
	completion.JSON = jsonStr
	return completion, nil
}

// requestAnthropic handles API requests to Anthropic's Claude models.
// Parameters:
//   - ctx: Context
//   - client: Configured Anthropic API client
//   - provider: The service provider (Anthropic)
//   - model: Claude model identifier
//   - prompt: The input prompt
//   - image: Optional image data for vision models
//   - betas: Optional beta features to enable
//
// Returns:
//   - *types.JSONCompletion: The API response and metadata
//   - error: Any API or processing errors
func requestAnthropic(
	ctx context.Context, client *anthropic.Client, provider types.LLMProvider, model, prompt string, image []byte, betas *[]anthropic.AnthropicBeta,
) (*types.JSONCompletion, error) {
	completion := &types.JSONCompletion{
		LLMCompletionMeta: types.LLMCompletionMeta{
			InputTokens:  0,
			OutputTokens: 0,
			Provider:     provider,
			Model:        model,
		},
	}

	messageContent := []anthropic.BetaContentBlockParamUnion{
		anthropic.BetaTextBlockParam{
			Type: anthropic.F(anthropic.BetaTextBlockParamTypeText),
			Text: anthropic.String(prompt),
		},
	}
	if image != nil {
		messageContent = append(messageContent, anthropic.BetaImageBlockParam{
			Type: anthropic.F(anthropic.BetaImageBlockParamTypeImage),
			Source: anthropic.F[anthropic.BetaImageBlockParamSourceUnion](
				anthropic.BetaImageBlockParamSource{
					Type:      anthropic.F(anthropic.BetaImageBlockParamSourceTypeBase64),
					MediaType: anthropic.F(anthropic.BetaImageBlockParamSourceMediaTypeImagePNG),
					Data:      anthropic.F(base64.StdEncoding.EncodeToString(image)),
				},
			),
		})
	}

	params := anthropic.BetaMessageNewParams{
		Model:     anthropic.F(model),
		MaxTokens: anthropic.F(int64(1024)),
		Messages: anthropic.F([]anthropic.BetaMessageParam{
			{
				Role:    anthropic.F(anthropic.BetaMessageParamRoleUser),
				Content: anthropic.F(messageContent),
			},
			{
				Role: anthropic.F(anthropic.BetaMessageParamRoleAssistant),
				Content: anthropic.F([]anthropic.BetaContentBlockParamUnion{
					anthropic.BetaTextBlockParam{
						Type: anthropic.F(anthropic.BetaTextBlockParamTypeText),
						Text: anthropic.String("```json"), // This forces the assistant to respond with a JSON block
					},
				}),
			},
		}),
		StopSequences: anthropic.F([]string{"```"}),
	}
	if betas != nil {
		params.Betas = anthropic.F(*betas)
	}

	response, err := client.Beta.Messages.New(ctx, params)
	if err != nil {
		return completion, fmt.Errorf("failed to get completion from %v: %w", provider, err)
	}

	completion.InputTokens = int(response.Usage.InputTokens)
	completion.OutputTokens = int(response.Usage.OutputTokens)

	jsonStr, err := utils.ParseJSON(response.Content[0].Text)
	if err != nil {
		return completion, fmt.Errorf("failed to parse JSON: %w", err)
	}
	completion.JSON = jsonStr
	return completion, nil
}
