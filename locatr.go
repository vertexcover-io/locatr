package locatr

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

//go:embed meta/htmlMinifier.js
var HTML_MINIFIER_JS_CONTENT string

type llmLocatorOutputDto struct {
	LocatorID          string `json:"locator_id"`
	completionResponse chatCompletionResponse
	Error              string `json:"error"`
}

type locatrOutputDto struct {
	llmLocatorOutputDto
	AttemptNo                int       `json:"attempt_no"`
	LocatrRequestInitiatedAt time.Time `json:"request_initiated_at"`
	LocatrRequestCompletedAt time.Time `json:"request_completed_at"`
}

type locatrResult struct {
	LocatrDescription        string    `json:"locatr_description"`
	Url                      string    `json:"url"`
	CacheHit                 bool      `json:"cache_hit"`
	Locatr                   string    `json:"locatr"`
	InputTokens              int       `json:"input_tokens"`
	OutputTokens             int       `json:"output_tokens"`
	TotalTokens              int       `json:"total_tokens"`
	LlmErrorMessage          string    `json:"llm_error_message"`
	ChatCompletionTimeTaken  int       `json:"llm_locatr_generation_time_taken"`
	AttemptNo                int       `json:"attempt_no"`
	LocatrRequestInitiatedAt time.Time `json:"request_initiated_at"`
	LocatrRequestCompletedAt time.Time `json:"request_completed_at"`
	AllLocatrs               []string  `json:"all_locatrs"`
}

func (l *locatrResult) MarshalJSON() ([]byte, error) {
	type Alias locatrResult
	return json.Marshal(&struct {
		*Alias
		LocatrRequestInitiatedAt string `json:"request_initiated_at"`
		LocatrRequestCompletedAt string `json:"request_completed_at"`
	}{
		Alias:                    (*Alias)(l),
		LocatrRequestInitiatedAt: l.LocatrRequestInitiatedAt.Format(time.RFC3339),
		LocatrRequestCompletedAt: l.LocatrRequestCompletedAt.Format(time.RFC3339),
	})
}

type cachedLocatrsDto struct {
	LocatrName string   `json:"locatr_name"`
	Locatrs    []string `json:"locatrs"`
}

type BaseLocatr struct {
	plugin        PluginInterface
	llmClient     LlmClientInterface
	reRankClient  ReRankInterface
	options       BaseLocatrOptions
	cachedLocatrs map[string][]cachedLocatrsDto
	initialized   bool
	logger        logInterface
	locatrResults []locatrResult
}

// BaseLocatrOptions is a struct that holds all the options for the locatr package
type BaseLocatrOptions struct {
	// CachePath is the path to the cache file
	CachePath string
	// UseCache is a flag to enable/disable cache
	UseCache bool
	// LogConfig is the log configuration
	LogConfig LogConfig

	// LocatrResultsFilePath is the path to the file where the locatr results will be written
	// If not provided, the results will be written to DEFAULT_LOCATR_RESULTS_FILE
	ResultsFilePath string

	// LLmClient is the client to interact with LLM
	LlmClient LlmClientInterface

	// ReRankClient is the client to interact with ReRank
	ReRankClient ReRankInterface
}

// NewBaseLocatr creates a new instance of BaseLocatr
// plugin: (playwright, puppeteer, etc)
// llmClient: struct that are returned by NewLlmClient
// options: All the options for the locatr package
func NewBaseLocatr(plugin PluginInterface, options BaseLocatrOptions) *BaseLocatr {
	if len(options.CachePath) == 0 {
		options.CachePath = DEFAULT_CACHE_PATH
	}
	if len(options.ResultsFilePath) == 0 {
		options.ResultsFilePath = DEFAULT_LOCATR_RESULTS_PATH
	}
	if options.LogConfig.Writer == nil {
		options.LogConfig.Writer = DefaultLogWriter
	}
	locatr := &BaseLocatr{
		plugin:        plugin,
		options:       options,
		cachedLocatrs: make(map[string][]cachedLocatrsDto),
		initialized:   false,
		logger:        NewLogger(options.LogConfig),
		locatrResults: []locatrResult{},
		reRankClient:  options.ReRankClient,
	}
	if options.LlmClient == nil {
		client, err := createLlmClientFromEnv()
		if err != nil {
			locatr.logger.Error(fmt.Sprintf("Failed to create LLM client: %v", err))
			return nil
		}
		locatr.llmClient = client
	} else {
		locatr.llmClient = options.LlmClient
	}
	return locatr
}

// getLocatorStr returns the locator string for the given user request
func (l *BaseLocatr) getLocatorStr(userReq string) (string, error) {
	if err := l.plugin.evaluateJsScript(HTML_MINIFIER_JS_CONTENT); err != nil {
		return "", ErrUnableToLoadJsScripts
	}
	l.initializeState()
	l.logger.Info(fmt.Sprintf("Getting locator for user request: `%s`", userReq))
	currentUrl := l.getCurrentUrl()
	locatr, err := l.loadLocatrsFromCache(userReq)
	if err == nil {
		return locatr, nil
	}
	l.logger.Info("Cache miss, starting dom minification")
	minifiedDOM, locatorsMap, err := l.getMinifiedDomAndLocatorMap()
	if err != nil {
		l.logger.Error(fmt.Sprintf("Failed to minify DOM and extract ID locator map: %v", err))
		return "", ErrUnableToMinifyHtmlDom
	}

	l.logger.Info("Extracting element ID using LLM")
	llmOutputs, err := l.locateElementId(minifiedDOM.ContentStr(), userReq)
	if err != nil {
		l.logger.Error(fmt.Sprintf("Failed to locate element ID: %v", err))
		if len(llmOutputs) > 0 {
			l.locatrResults = append(l.locatrResults,
				createLocatrResultFromOutput(
					userReq, "", currentUrl, llmOutputs,
				)...,
			)
		}
		return "", ErrUnableToLocateElementId
	}

	locators, ok := (*locatorsMap)[llmOutputs[len(llmOutputs)-1].LocatorID]
	if !ok {
		l.logger.Error("Invalid element ID generated")
		return "", ErrInvalidElementIdGenerated
	}

	validLocator, err := l.getValidLocator(locators)
	if err != nil {
		l.logger.Error(fmt.Sprintf("Failed to find valid locator: %v", err))
		return "", ErrUnableToFindValidLocator
	}
	l.locatrResults = append(l.locatrResults,
		createLocatrResultFromOutput(
			userReq, validLocator, currentUrl, llmOutputs,
		)...,
	)
	if l.options.UseCache {
		l.logger.Info(fmt.Sprintf("Adding locatrs of `%s to cache", userReq))
		l.logger.Debug(fmt.Sprintf("Adding Locars of `%s`: `%v` to cache", userReq, locators))
		l.addCachedLocatrs(currentUrl, userReq, locators)
		value, err := json.Marshal(l.cachedLocatrs)
		if err != nil {
			l.logger.Error(fmt.Sprintf("Failed to marshal cache: %v", err))
			return "", fmt.Errorf("%w: %v", ErrFailedToMarshalJson, err)
		}
		if err = writeLocatorsToCache(l.options.CachePath, value); err != nil {
			l.logger.Error(fmt.Sprintf("Failed to write cache: %v", err))
			return "", fmt.Errorf("%w: %v", ErrFailedToWriteCache, err)
		}
	}
	return validLocator, nil

}
func (l *BaseLocatr) getCurrentUrl() string {
	if value, err := l.plugin.evaluateJsFunction("window.location.href"); err == nil {
		return value
	} else {
		l.logger.Error(fmt.Sprintf("Failed to get current URL: %v", err))
	}
	return ""
}

func (l *BaseLocatr) getMinifiedDomAndLocatorMap() (*ElementSpec, *IdToLocatorMap, error) {
	result, _ := l.plugin.evaluateJsFunction("minifyHTML()")
	elementSpec := &ElementSpec{}
	if err := json.Unmarshal([]byte(result), elementSpec); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal ElementSpec json: %v", err)
	}

	result, _ = l.plugin.evaluateJsFunction("mapElementsToJson()")
	idLocatorMap := &IdToLocatorMap{}
	if err := json.Unmarshal([]byte(result), idLocatorMap); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal IdToLocatorMap json: %v", err)
	}

	return elementSpec, idLocatorMap, nil
}

func (l *BaseLocatr) getValidLocator(locators []string) (string, error) {
	for _, locator := range locators {
		if value, _ := l.plugin.evaluateJsFunction(fmt.Sprintf("isValidLocator('%s')", locator)); value == "true" {
			l.logger.Debug(fmt.Sprintf("Valid locator found: `%s`", locator))
			return locator, nil
		}
	}
	return "", ErrUnableToFindValidLocator
}
func (l *BaseLocatr) getReRankedChunks(htmlDom string, userReq string) ([]string, error) {
	chunks := SplitHtml(htmlDom, HTML_SEPARATORS, CHUNK_SIZE)
	l.logger.Debug(fmt.Sprintf("SplitHtml resulted in %d chunks.", len(chunks)))
	request := reRankRequest{
		Query:     userReq,
		Documents: chunks,
	}
	reRankResults, err := l.reRankClient.reRank(request)
	if err != nil {
		l.logger.Error(fmt.Sprintf("Failed to re-rank chunks: %v", err))
		return nil, fmt.Errorf("failed to re-rank chunks: %v", err)
	}
	for _, result := range *reRankResults {
		l.logger.Debug(fmt.Sprintf("Re-rank result index: %d, score: %f", result.Index, result.Score))
	}
	return sortRerankChunks(chunks, *reRankResults), nil
}
func (l *BaseLocatr) llmGetElementId(htmlDom string, userReq string) (*llmLocatorOutputDto, error) {
	jsonData, err := json.Marshal(&llmWebInputDto{
		HtmlDom: htmlDom,
		UserReq: userReq,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal web input json: %v", err)
	}

	prompt := fmt.Sprintf("%s%s", LOCATR_PROMPT, string(jsonData))

	llmResponse, err := l.llmClient.ChatCompletion(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to get response from LLM: %v", err)
	}

	llmLocatorOutput := &llmLocatorOutputDto{
		completionResponse: *llmResponse,
	}

	l.logger.Debug(fmt.Sprintf("LLM response: %s", llmResponse.Completion))

	llmResponse.Completion = fixLLmJson(llmResponse.Completion)

	l.logger.Debug(fmt.Sprintf("Repaired LLM response: %s", llmResponse.Completion))

	if err = json.Unmarshal([]byte(llmResponse.Completion), llmLocatorOutput); err != nil {
		return nil, fmt.Errorf("failed to unmarshal llmLocatorOutputDto json: %v", err)
	}
	return llmLocatorOutput, nil
}
func (l *BaseLocatr) getLocatrOutput(htmlDOM string, userReq string) (*locatrOutputDto, error) {
	result, err := l.llmGetElementId(htmlDOM, userReq)
	if err != nil {
		return nil, err
	}
	endAt := time.Now()
	if result.Error == "" {
		return &locatrOutputDto{
			llmLocatorOutputDto:      *result,
			LocatrRequestCompletedAt: endAt,
		}, nil
	}
	return nil, ErrLocatrRetrievalFailed
}
func (l *BaseLocatr) locateElementId(htmlDOM string, userReq string) ([]locatrOutputDto, error) {
	llmOutputs := []locatrOutputDto{}
	requestInitiatedAt := time.Now()
	if l.reRankClient == nil {
		l.logger.Debug("No rerank client setup sending full dom to llm.")
		result, err := l.getLocatrOutput(htmlDOM, userReq)
		if err != nil {
			return llmOutputs, err
		}
		result.LocatrRequestInitiatedAt = requestInitiatedAt
		llmOutputs = append(llmOutputs, *result)
		if result.Error == "" {
			return llmOutputs, nil
		}
		return llmOutputs, ErrLocatrRetrievalFailed
	}
	chunks, err := l.getReRankedChunks(htmlDOM, userReq)
	if err != nil {
		return llmOutputs, err
	}
	if len(chunks) == 0 {
		l.logger.Debug("No chunks to process")
		return llmOutputs, ErrNoChunksToProcess
	}
	if len(chunks) == 1 {
		l.logger.Debug("Only one chunk to process, sending to llm.")
		result, err := l.getLocatrOutput(htmlDOM, userReq)
		if err != nil {
			return llmOutputs, err
		}
		result.LocatrRequestInitiatedAt = requestInitiatedAt
		llmOutputs = append(llmOutputs, *result)
		if result.Error == "" {
			return llmOutputs, nil
		}
		return llmOutputs, ErrLocatrRetrievalFailed
	}

	var domToProcess string

	for attempt := 0; attempt < MAX_RETRIES_WITH_RERANK; attempt++ {
		switch attempt {
		case 0:
			domToProcess = strings.Join(chunks[0:MAX_RETRIES_WITH_RERANK], "\n")
		case 1:
			endIndex := MAX_CHUNKS_EACH_RERANK_ITERATION * 2
			if endIndex > len(chunks) {
				endIndex = len(chunks)
				attempt++
				l.logger.Debug(fmt.Sprintf("Max chunks reached in attempt %d, this will be the final attempt.", attempt+1))
			}
			domToProcess = strings.
				Join(chunks[MAX_CHUNKS_EACH_RERANK_ITERATION:endIndex], "\n")
		default:
			domToProcess = htmlDOM
		}

		l.logger.Debug(fmt.Sprintf("attempt no (%d) to find locatr with reranking", attempt+1))
		requestCompletedAt := time.Now()

		result, err := l.llmGetElementId(domToProcess, userReq)
		if err != nil {
			return llmOutputs, err
		}

		llmOutputs = append(llmOutputs, locatrOutputDto{
			llmLocatorOutputDto:      *result,
			AttemptNo:                attempt,
			LocatrRequestInitiatedAt: requestInitiatedAt,
			LocatrRequestCompletedAt: requestCompletedAt,
		})

		if result.Error == "" {
			return llmOutputs, nil
		}

		l.logger.Error(fmt.Sprintf("Failed to get locatr in %d attempt(s) : %s", attempt+1, result.Error))
	}
	return llmOutputs, ErrLocatrRetrievalAttemptsExhausted
}

func (l *BaseLocatr) writeLocatrResultsToFile() {
	l.logger.Info(fmt.Sprintf("Writing locatr results to file: %s", l.options.ResultsFilePath))
	file, err := os.OpenFile(l.options.ResultsFilePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		l.logger.Error(fmt.Sprintf("Failed to create file locatr results file: %v", err))
		return
	}
	defer file.Close()
	l.logger.Debug(fmt.Sprintf("Results to write: %v", l.locatrResults))
	value, err := json.Marshal(l.locatrResults)
	if err != nil {
		l.logger.Error(fmt.Sprintf("Failed to marshal locatr results json: %v", err))
		return
	}
	if _, err := file.Write(value); err != nil {
		l.logger.Error(fmt.Sprintf("Failed to write locatr results to file: %v", err))
	} else {
		l.logger.Info(fmt.Sprintf("Results written to file: %s", l.options.ResultsFilePath))
	}
}

func (l *BaseLocatr) getLocatrResults() []locatrResult {
	return l.locatrResults
}
