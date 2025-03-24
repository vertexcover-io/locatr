package mode

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/vertexcover-io/locatr/golang/internal/constants"
	"github.com/vertexcover-io/locatr/golang/internal/splitters"
	"github.com/vertexcover-io/locatr/golang/internal/utils"
	"github.com/vertexcover-io/locatr/golang/logging"
	"github.com/vertexcover-io/locatr/golang/types"
)

// VISUAL_ANALYSIS_PROMPT_TEMPLATE defines the system prompt for identifying coordinates in screenshots.
// The prompt guides the LLM to determine precise (X, Y) coordinates for UI elements in the given resolution
// screenshot based on user requests and element types (buttons, text fields, etc.).
const VISUAL_ANALYSIS_PROMPT_TEMPLATE string = `Your task is to identify the exact (X, Y) coordinates for a described element or area on a screenshot of a web page with a resolution of %d x %d.

Analyze the screenshot and the user's request carefully to determine the appropriate coordinates. The coordinates should point to the center of the described element when possible.

Guidelines for coordinate identification:
1. For buttons, links, and clickable elements: Target the center of the element
2. For text fields: Target the beginning of the input area
3. For larger areas: Target the most relevant point that satisfies the user's intent

If you cannot confidently determine the coordinates based on the provided information, return an empty string for the point and provide a helpful error message explaining why.

Provide your response in valid JSON format with the following structure:
{
    "element_point": "x, y",  // Comma-separated X and Y coordinates, or empty string if coordinates cannot be determined
    "error": ""       // A descriptive error message if coordinates cannot be determined, otherwise an empty string
}

User request: %s
Be precise in your coordinate estimation as these will be used for automated interactions.
`

type VisualAnalysisMode struct {
	// Resolution to use for viewport size, defaults to 1280x800
	Resolution *types.Resolution `json:"resolution"`
	// Maximum number of relevant screenshots to use for analysis. Defaults to constants.DEFAULT_TOP_N
	MaxAttempts int `json:"max_attempts"`
}

func (m *VisualAnalysisMode) ProcessRequest(
	request string,
	plugin types.PluginInterface,
	llmClient types.LLMClientInterface,
	rerankerClient types.RerankerClientInterface,
	logger *slog.Logger,
	completion *types.LocatrCompletion,
) error {
	defer logging.CreateTopic("[Mode] Visual Analysis", logger)()
	m.applyDefaults()
	dom, err := plugin.GetMinifiedDOM()
	if err != nil {
		return err
	}
	domChunks := splitters.SplitHtml(
		dom.RootElement.Repr(), constants.HTML_SEPARATORS, constants.DEFAULT_CHUNK_SIZE,
	)

	results, err := rerankerClient.Rerank(
		&types.RerankRequest{
			Query: request, Documents: domChunks, TopN: m.MaxAttempts,
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
		ElementPoint string `json:"element_point"`
		ErrorMessage string `json:"error"`
	}

	for attempt, chunk := range domChunks {
		logger.Info("Attempt number", "attempt", attempt+1)

		id, err := utils.ExtractFirstUniqueID(chunk)
		if err != nil {
			continue
		}
		if err := plugin.SetViewportSize(m.Resolution.Width, m.Resolution.Height); err != nil {
			logger.Error("couldn't set viewport size", "error", err)
			continue
		}

		locator := locatorMap[id][0]
		chunkLocation, err := plugin.GetElementLocation(locator)
		if err != nil {
			logger.Error("couldn't find chunk on the page", "error", err)
			continue
		}
		screenshotBytes, err := plugin.TakeScreenshot()
		if err != nil {
			logger.Error("couldn't take screenshot", "error", err)
			continue
		}

		prompt := fmt.Sprintf(
			VISUAL_ANALYSIS_PROMPT_TEMPLATE,
			m.Resolution.Width,
			m.Resolution.Height,
			request,
		)

		jsonCompletion, err := llmClient.GetJSONCompletion(prompt, screenshotBytes)
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

		if strings.TrimSpace(analysisOutput.ErrorMessage) != "" {
			logger.Error("error getting relevant element point", "error", analysisOutput.ErrorMessage)
			continue
		}

		if strings.TrimSpace(analysisOutput.ElementPoint) == "" {
			logger.Error("no relevant element point found")
			continue
		}

		point := strings.Split(analysisOutput.ElementPoint, ",")
		if len(point) != 2 {
			logger.Error("invalid point format, expected format: x,y")
			continue
		}

		xCoord, err := strconv.ParseFloat(strings.TrimSpace(point[0]), 64)
		if err != nil {
			logger.Error("invalid x coordinate")
			continue
		}

		yCoord, err := strconv.ParseFloat(strings.TrimSpace(point[1]), 64)
		if err != nil {
			logger.Error("invalid y coordinate")
			continue
		}

		elementPoint := types.Point{X: xCoord, Y: yCoord}
		logger.Info("elementPoint", "point", elementPoint)
		locators, err := plugin.GetElementLocators(&types.Location{
			Point:          elementPoint,
			ScrollPosition: chunkLocation.ScrollPosition,
		})
		if err != nil {
			logger.Error("couldn't get element locators", "error", err)
			continue
		}
		if len(locators) == 0 {
			logger.Error("no element found at point", "point", elementPoint)
			continue
		}
		completion.Locators = locators
		completion.LocatorType = dom.Metadata.LocatorType
		return nil
	}
	return errors.New("no relevant element point found in the DOM")
}

func (m *VisualAnalysisMode) applyDefaults() {
	if m.Resolution == nil {
		m.Resolution = &types.Resolution{
			Width:  1280,
			Height: 800,
		}
	}
	if m.MaxAttempts <= 0 {
		m.MaxAttempts = constants.DEFAULT_TOP_N
	}
}
