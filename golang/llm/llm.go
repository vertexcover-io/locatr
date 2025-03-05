package llm

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	anthropicOption "github.com/anthropics/anthropic-sdk-go/option"
	"github.com/kaptinlin/jsonrepair"
	"github.com/openai/openai-go"
	openaiOption "github.com/openai/openai-go/option"
	"github.com/vertexcover-io/locatr/golang/types"
)

// LLMProvider constants define the supported Language Model service providers
const (
	OpenAI     types.LLMProvider = "openai"      // OpenAI API provider (e.g., GPT-4, GPT-3.5)
	Anthropic  types.LLMProvider = "anthropic"   // Anthropic API provider (e.g., Claude models)
	Groq       types.LLMProvider = "groq"        // Groq API provider
	OpenRouter types.LLMProvider = "open-router" // OpenRouter API aggregation service
)

// GET_ID_FROM_DOM_PROMPT_TEMPLATE defines the system prompt for extracting element IDs from HTML DOM.
// The prompt instructs the LLM to identify HTML elements based on user requirements and supported interactions
// (clickable, hoverable, inputable, selectable) by analyzing data-supported-primitives attributes.
const GET_ID_FROM_DOM_PROMPT_TEMPLATE string = `Your task is to identify the HTML element that matches a user's requirement from a given HTML DOM structure and return its unique_id in a JSON format. If the element is not found, provide an appropriate error message in the JSON output.

Each HTML element may contain an attribute called "data-supported-primitives" which indicates its supported interactions. The following attributes determine whether an element is "clickable", "hoverable", "inputable", or "selectable":

1. "clickable": The element supports click interactions and will have "data-supported-primitives" set to "click".
2. "hoverable": The element supports hover interactions and will have "data-supported-primitives" set to "hover".
3. "inputable": The element supports text input interactions and will have "data-supported-primitives" set to "input_text". If this attribute is not present then the input is read-only.
4. "selectable": The element supports selecting options and will have "data-supported-primitives" set to "select_option".

Provide your response in valid JSON format with the following structure:
{
  "locator_id": "str",     // The unique_id of the element that matches the user's requirement.
  "error": "str"           // An appropriate error message if the element is not found.
}

Input:
{
  "html_dom": "%s",
  "user_req": "%s"
}
Process the input accordingly and ensure that if the element is not found, the "error" field contains a relevant message.
`

// GET_POINT_FROM_SCREENSHOT_PROMPT_TEMPLATE defines the system prompt for identifying coordinates in screenshots.
// The prompt guides the LLM to determine precise (X, Y) coordinates for UI elements in a 1280x800 resolution
// screenshot based on user descriptions and element types (buttons, text fields, etc.).
const GET_POINT_FROM_SCREENSHOT_PROMPT_TEMPLATE string = `Your task is to identify the exact (X, Y) coordinates for a described element or area on a screenshot of a web page with a resolution of 1280x800.

Analyze the screenshot and the user's description carefully to determine the appropriate coordinates. The coordinates should point to the center of the described element when possible.

Guidelines for coordinate identification:
1. For buttons, links, and clickable elements: Target the center of the element
2. For text fields: Target the beginning of the input area
3. For larger areas: Target the most relevant point that satisfies the user's intent

If you cannot confidently determine the coordinates based on the provided information, return an empty string for the point and provide a helpful error message explaining why.

Provide your response in valid JSON format with the following structure:
{
    "point": "x, y",  // Comma-separated X and Y coordinates, or empty string if coordinates cannot be determined
    "error": ""       // A descriptive error message if coordinates cannot be determined, otherwise an empty string
}

Description: %s
Be precise in your coordinate estimation as these will be used for automated interactions.
`

// LLMClient represents a client for interacting with Language Model APIs.
// It encapsulates the provider configuration and completion request handling.
type LLMClient struct {
	provider         types.LLMProvider
	model            string
	getRawCompletion func(prompt string, image []byte) (*types.RawCompletion, error)
}

// NewLLMClient creates a new LLMClient instance with the specified configuration.
// Parameters:
//   - provider: The LLM service provider (OpenAI, Anthropic, Groq, or OpenRouter)
//   - model: The specific model name to use (e.g., "gpt-4", "claude-3-sonnet")
//   - apiKey: Authentication key for the chosen provider
//
// Returns:
//   - *LLMClient: Configured client instance
//   - error: Any initialization errors
func NewLLMClient(provider types.LLMProvider, model string, apiKey string) (*LLMClient, error) {
	client := &LLMClient{provider: provider, model: model}

	switch provider {
	case OpenAI:
		client.getRawCompletion = func(prompt string, image []byte) (*types.RawCompletion, error) {
			return requestOpenAI(
				openai.NewClient(openaiOption.WithAPIKey(apiKey)),
				client.provider,
				client.model,
				prompt,
				image,
			)
		}
	case Anthropic:
		betas := []anthropic.AnthropicBeta{}
		if strings.HasPrefix(model, "claude-3-5-sonnet") {
			betas = append(betas, anthropic.AnthropicBetaComputerUse2024_10_22)
		}
		if strings.HasPrefix(model, "claude-3-7-sonnet") {
			betas = append(betas, anthropic.AnthropicBetaComputerUse2025_01_24)
		}
		client.getRawCompletion = func(prompt string, image []byte) (*types.RawCompletion, error) {
			return requestAnthropic(
				anthropic.NewClient(anthropicOption.WithAPIKey(apiKey)),
				client.provider,
				client.model,
				prompt,
				image,
				&betas,
			)
		}
	case Groq:
		client.getRawCompletion = func(prompt string, image []byte) (*types.RawCompletion, error) {
			return requestOpenAI(
				openai.NewClient(
					openaiOption.WithBaseURL("https://openrouter.ai/api/v1/"),
					openaiOption.WithAPIKey(apiKey),
				),
				client.provider,
				client.model,
				prompt,
				image,
			)
		}
	case OpenRouter:
		client.getRawCompletion = func(prompt string, image []byte) (*types.RawCompletion, error) {
			return requestOpenAI(
				openai.NewClient(
					openaiOption.WithBaseURL("https://openrouter.ai/api/v1/"),
					openaiOption.WithAPIKey(apiKey),
				),
				client.provider,
				client.model,
				prompt,
				image,
			)
		}
	default:
		return nil, errors.New("invalid provider for llm")
	}
	return client, nil
}

// CreateLLMClientFromEnv initializes an LLMClient using environment variables:
//   - LLM_PROVIDER: The service provider name
//   - LLM_MODEL: The model identifier
//   - LLM_API_KEY: Authentication key for the provider
//
// Returns:
//   - *LLMClient: Configured client instance
//   - error: Any initialization errors
func CreateLLMClientFromEnv() (*LLMClient, error) {
	provider := os.Getenv("LLM_PROVIDER")
	if provider == "" {
		return nil, errors.New("invalid provider for llm")
	}
	model := os.Getenv("LLM_MODEL")
	if model == "" {
		return nil, errors.New("model name is required")
	}
	apiKey := os.Getenv("LLM_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("'%s' associated API key is required", provider)
	}
	return NewLLMClient(types.LLMProvider(provider), model, apiKey)
}

// GetIdCompletion analyzes HTML DOM structure to find elements matching user requirements.
// Parameters:
//   - userRequest: Natural language description of the desired element
//   - dom: HTML DOM structure as a string
//
// Returns:
//   - *types.IdFromDOMCompletion: Contains the matched element ID or error message
//   - error: Any processing errors
func (client *LLMClient) GetIdCompletion(userRequest string, dom string) (*types.IdFromDOMCompletion, error) {

	prompt := fmt.Sprintf(GET_ID_FROM_DOM_PROMPT_TEMPLATE, dom, userRequest)
	rawCompletion, err := client.getRawCompletion(prompt, nil)
	completion := &types.IdFromDOMCompletion{
		LLMCompletionMeta: rawCompletion.LLMCompletionMeta,
	}
	if err != nil {
		return completion, err
	}

	repaired, err := jsonrepair.JSONRepair(extractJSON(rawCompletion.Text))
	if err != nil {
		return completion, fmt.Errorf("failed to repair JSON: %w", err)
	}

	var output struct {
		Id    string `json:"locator_id"`
		Error string `json:"error"`
	}

	if err = json.Unmarshal([]byte(repaired), &output); err != nil {
		return completion, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Check if there's an error message
	if strings.TrimSpace(output.Error) != "" {
		completion.ErrorMessage = output.Error
		return completion, nil
	}

	if strings.TrimSpace(output.Id) == "" {
		return completion, errors.New("no Id found")
	}

	completion.Id = output.Id
	return completion, nil
}

// GetPointCompletion analyzes a screenshot to determine coordinates for described elements.
// Currently only supported by Anthropic's Claude models.
// Parameters:
//   - userRequest: Natural language description of the target element/area
//   - screenshot: Image bytes of the webpage screenshot
//
// Returns:
//   - *types.PointFromScreenshotCompletion: Contains the coordinates or error message
//   - error: Any processing errors
func (client *LLMClient) GetPointCompletion(userRequest string, screenshot []byte) (*types.PointFromScreenshotCompletion, error) {
	if client.provider != Anthropic {
		return &types.PointFromScreenshotCompletion{
			LLMCompletionMeta: types.LLMCompletionMeta{
				TimeTaken:    0,
				InputTokens:  0,
				OutputTokens: 0,
				TotalTokens:  0,
				Provider:     client.provider,
				Model:        client.model,
			},
		}, errors.New("as of now, OpenAI and compatible providers does not support grounding. Please use Anthropic sonnet models")
	}
	prompt := fmt.Sprintf(GET_POINT_FROM_SCREENSHOT_PROMPT_TEMPLATE, userRequest)
	rawCompletion, err := client.getRawCompletion(prompt, screenshot)
	completion := &types.PointFromScreenshotCompletion{
		LLMCompletionMeta: rawCompletion.LLMCompletionMeta,
	}
	if err != nil {
		return completion, err
	}

	// Parse the JSON response
	repaired, err := jsonrepair.JSONRepair(extractJSON(rawCompletion.Text))
	if err != nil {
		return completion, fmt.Errorf("failed to repair JSON: %w", err)
	}

	var output struct {
		Point string `json:"point"`
		Error string `json:"error"`
	}

	if err = json.Unmarshal([]byte(repaired), &output); err != nil {
		return completion, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Check if there's an error message
	if strings.TrimSpace(output.Error) != "" {
		completion.ErrorMessage = output.Error
		return completion, nil
	}

	if strings.TrimSpace(output.Point) == "" {
		return completion, errors.New("no point found")
	}

	point := strings.Split(output.Point, ",")
	if len(point) != 2 {
		return completion, errors.New("invalid point format")
	}

	xCoord, err := strconv.ParseFloat(strings.TrimSpace(point[0]), 64)
	if err != nil {
		return completion, errors.New("invalid x coordinate")
	}

	yCoord, err := strconv.ParseFloat(strings.TrimSpace(point[1]), 64)
	if err != nil {
		return completion, errors.New("invalid y coordinate")
	}

	completion.Point = types.Point{X: xCoord, Y: yCoord}
	return completion, nil
}

// GetProvider returns the configured LLM service provider for this client.
func (client *LLMClient) GetProvider() types.LLMProvider {
	return client.provider
}

// GetModel returns the configured model name for this client.
func (client *LLMClient) GetModel() string {
	return client.model
}

// requestOpenAI handles API requests to OpenAI-compatible endpoints (OpenAI, Groq, OpenRouter).
// Parameters:
//   - client: Configured OpenAI API client
//   - provider: The service provider being used
//   - model: Model identifier
//   - prompt: The input prompt
//   - image: Optional image data for vision models
//
// Returns:
//   - *types.RawCompletion: The API response and metadata
//   - error: Any API or processing errors
func requestOpenAI(client *openai.Client, provider types.LLMProvider, model, prompt string, image []byte) (*types.RawCompletion, error) {
	completion := &types.RawCompletion{
		LLMCompletionMeta: types.LLMCompletionMeta{
			TimeTaken:    0,
			InputTokens:  0,
			OutputTokens: 0,
			TotalTokens:  0,
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

	start := time.Now()
	resp, err := client.Chat.Completions.New(
		context.Background(), openai.ChatCompletionNewParams{
			Model:    openai.F(model),
			Messages: openai.F(messages),
			ResponseFormat: openai.F[openai.ChatCompletionNewParamsResponseFormatUnion](
				openai.ResponseFormatJSONObjectParam{
					Type: openai.F(openai.ResponseFormatJSONObjectTypeJSONObject),
				},
			),
		},
	)
	completion.TimeTaken = int(time.Since(start).Seconds())

	if err != nil {
		return completion, fmt.Errorf("failed to get completion from %v: %w", provider, err)
	}

	completion.Text = resp.Choices[0].Message.Content
	completion.InputTokens = int(resp.Usage.PromptTokens)
	completion.OutputTokens = int(resp.Usage.CompletionTokens)
	completion.TotalTokens = completion.InputTokens + completion.OutputTokens
	return completion, nil
}

// requestAnthropic handles API requests to Anthropic's Claude models.
// Parameters:
//   - client: Configured Anthropic API client
//   - provider: The service provider (Anthropic)
//   - model: Claude model identifier
//   - prompt: The input prompt
//   - image: Optional image data for vision models
//   - betas: Optional beta features to enable
//
// Returns:
//   - *types.RawCompletion: The API response and metadata
//   - error: Any API or processing errors
func requestAnthropic(
	client *anthropic.Client, provider types.LLMProvider, model, prompt string, image []byte, betas *[]anthropic.AnthropicBeta,
) (*types.RawCompletion, error) {
	completion := &types.RawCompletion{
		LLMCompletionMeta: types.LLMCompletionMeta{
			TimeTaken:    0,
			InputTokens:  0,
			OutputTokens: 0,
			TotalTokens:  0,
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
			Source: anthropic.F[anthropic.BetaImageBlockParamSourceUnion](anthropic.BetaImageBlockParamSource{
				Type:      anthropic.F(anthropic.BetaImageBlockParamSourceTypeBase64),
				MediaType: anthropic.F(anthropic.BetaImageBlockParamSourceMediaTypeImagePNG),
				Data:      anthropic.F(base64.StdEncoding.EncodeToString(image)),
			}),
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
						Text: anthropic.String("{"), // This forces the assistant to respond with a JSON object
					},
				}),
			},
		}),
		StopSequences: anthropic.F([]string{"}"}),
	}
	if betas != nil {
		params.Betas = anthropic.F(*betas)
	}

	start := time.Now()
	resp, err := client.Beta.Messages.New(context.TODO(), params)

	completion.TimeTaken = int(time.Since(start).Seconds())
	if err != nil {
		return completion, fmt.Errorf("failed to get completion from %v: %w", provider, err)
	}

	text := resp.Content[0].Text
	if !strings.HasPrefix(text, "{") {
		text = "{" + text // Add a leading { to the response for a valid JSON object
	}
	if !strings.HasSuffix(text, "}") {
		text += "}" // Add a trailing } to the response for a valid JSON object
	}
	completion.Text = text
	completion.InputTokens = int(resp.Usage.InputTokens)
	completion.OutputTokens = int(resp.Usage.OutputTokens)
	completion.TotalTokens = completion.InputTokens + completion.OutputTokens
	return completion, nil
}

// extractJSON removes markdown code block syntax from a JSON string.
// This helps clean up LLM responses that may include JSON within markdown formatting.
func extractJSON(json string) string {
	json = strings.TrimPrefix(json, "```")
	json = strings.TrimPrefix(json, "json")
	json = strings.TrimSuffix(json, "```")

	return json
}
