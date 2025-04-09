package mode

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/vertexcover-io/locatr/pkg/internal/constants"
	"github.com/vertexcover-io/locatr/pkg/internal/splitters"
	"github.com/vertexcover-io/locatr/pkg/internal/utils"
	"github.com/vertexcover-io/locatr/pkg/logging"
	"github.com/vertexcover-io/locatr/pkg/types"
)

// DOM_ANALYSIS_PROMPT_TEMPLATE defines the system prompt for extracting element IDs from DOM.
// The prompt instructs the LLM to identify elements based on user requirements and supported interactions
// (clickable, hoverable, inputable, selectable) by analyzing data-supported-primitives attributes.
const DOM_ANALYSIS_PROMPT_TEMPLATE string = `Your task is to identify the element that matches a user's requirement from a given DOM structure and return its unique_id in a JSON format. If the element is not found, provide an appropriate error message in the JSON output.

Each element may contain an attribute called "data-supported-primitives" which indicates its supported interactions. The following attributes determine whether an element is "clickable", "hoverable", "inputable", or "selectable":

1. "clickable": The element supports click interactions and will have "data-supported-primitives" set to "click".
2. "hoverable": The element supports hover interactions and will have "data-supported-primitives" set to "hover".
3. "inputable": The element supports text input interactions and will have "data-supported-primitives" set to "input_text". If this attribute is not present then the input is read-only.
4. "selectable": The element supports selecting options and will have "data-supported-primitives" set to "select_option".

Provide your response in valid JSON format with the following structure:
{
  "element_id": "str",     // The unique id of the element that matches the user's requirement.
  "error": "str"           // An appropriate error message if the element is not found.
}

Input:
{
  "dom": "%s",
  "user_request": "%s"
}
Process the input accordingly and ensure that if the element is not found, the "error" field contains a relevant message.
`

type DOMAnalysisMode struct {
	// The size of the chunks to process. Defaults to constants.DEFAULT_CHUNK_SIZE
	ChunkSize int `json:"chunk_size"`
	// The maximum number of attempts. Defaults to constants.DEFAULT_MAX_ATTEMPTS
	MaxAttempts int `json:"max_attempts"`
	// The number of chunks to process per attempt. Defaults to constants.DEFAULT_CHUNKS_PER_ATTEMPT
	ChunksPerAttempt int `json:"chunks_per_attempt"`
}

func (m *DOMAnalysisMode) ProcessRequest(
	request string,
	plugin types.PluginInterface,
	llmClient types.LLMClientInterface,
	rerankerClient types.RerankerClientInterface,
	logger *slog.Logger,
	completion *types.LocatrCompletion,
) error {
	defer logging.CreateTopic("[Mode] DOM Analysis", logger)()
	m.applyDefaults()
	dom, err := plugin.GetMinifiedDOM()
	if err != nil {
		return err
	}
	domChunks := splitters.SplitHtml(dom.RootElement.Repr(), constants.HTML_SEPARATORS, m.ChunkSize)

	results, err := rerankerClient.Rerank(
		&types.RerankRequest{
			Query: request, Documents: domChunks, TopN: m.MaxAttempts * m.ChunksPerAttempt,
		},
	)
	if err != nil {
		return err
	}
	domChunks = utils.SortRerankChunks(domChunks, results)
	logger.Info("Max chunks to process", "count", len(domChunks))
	if len(domChunks) == 0 {
		return fmt.Errorf("no chunks to process")
	}

	locatorMap := dom.Metadata.LocatorMap
	var analysisOutput struct {
		ElementId    string `json:"element_id"`
		ErrorMessage string `json:"error"`
	}

	for attempt := range m.MaxAttempts {
		startIndex := attempt * m.ChunksPerAttempt
		endIndex := startIndex + m.ChunksPerAttempt

		if startIndex >= len(domChunks) {
			break
		}

		if endIndex > len(domChunks) {
			endIndex = len(domChunks)
		}

		chunks := domChunks[startIndex:endIndex]
		logger.Info("Attempt number", "attempt", attempt+1)

		prompt := fmt.Sprintf(DOM_ANALYSIS_PROMPT_TEMPLATE, strings.Join(chunks, "\n"), request)
		jsonCompletion, err := llmClient.GetJSONCompletion(prompt, nil)
		completion.InputTokens += jsonCompletion.InputTokens
		completion.OutputTokens += jsonCompletion.OutputTokens
		if err != nil {
			logger.Error("couldn't get JSON completion", "error", err)
			continue
		}

		if err = json.Unmarshal([]byte(jsonCompletion.JSON), &analysisOutput); err != nil {
			logger.Error("failed to unmarshal JSON", "error", err)
			continue
		}

		// Check if there's an error message
		if strings.TrimSpace(analysisOutput.ErrorMessage) != "" {
			logger.Error("error getting relevant element ID", "error", analysisOutput.ErrorMessage)
			continue
		}

		if strings.TrimSpace(analysisOutput.ElementId) == "" {
			logger.Error("no relevant element ID found")
			continue
		}

		locators := locatorMap[analysisOutput.ElementId]
		if len(locators) == 0 {
			logger.Error("no locators found associated with element ID", "element_id", analysisOutput.ElementId)
			continue
		}
		completion.Locators = locators
		completion.LocatorType = dom.Metadata.LocatorType
		return nil
	}
	return errors.New("no relevant element ID found in the DOM")
}

func (m *DOMAnalysisMode) applyDefaults() {
	if m.ChunkSize <= 0 {
		m.ChunkSize = constants.DEFAULT_CHUNK_SIZE
	}
	if m.MaxAttempts <= 0 {
		m.MaxAttempts = constants.DEFAULT_MAX_ATTEMPTS
	}
	if m.ChunksPerAttempt <= 0 {
		m.ChunksPerAttempt = constants.DEFAULT_CHUNKS_PER_ATTEMPT
	}
}
