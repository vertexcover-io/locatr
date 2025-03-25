package locatr

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vertexcover-io/locatr/golang/elementSpec"
	"github.com/vertexcover-io/locatr/golang/llm"
	"github.com/vertexcover-io/locatr/golang/logger"
	"github.com/vertexcover-io/locatr/golang/reranker"
	"github.com/vertexcover-io/locatr/golang/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type SelectorType string

type PluginInterface interface {
	GetMinifiedDomAndLocatorMap(context.Context) (
		*elementSpec.ElementSpec,
		*elementSpec.IdToLocatorMap,
		SelectorType,
		error,
	)
	GetCurrentContext(context.Context) string
	IsValidLocator(ctx context.Context, locatr string) (bool, error)
}

type LocatrInterface interface {
	WriteResultsToFile()
	GetLocatrResults() []LocatrResult
	GetLocatrStr(ctx context.Context, userReq string) (*LocatrOutput, error)
}

type llmLocatorOutputDto struct {
	LocatorID          string `json:"locator_id"`
	completionResponse llm.ChatCompletionResponse
	Error              string `json:"error"`
}

type locatrOutputDto struct {
	llmLocatorOutputDto
	AttemptNo                int       `json:"attempt_no"`
	LocatrRequestInitiatedAt time.Time `json:"request_initiated_at"`
	LocatrRequestCompletedAt time.Time `json:"request_completed_at"`
}

type LocatrResult struct {
	LocatrDescription        string       `json:"locatr_description"`
	Url                      string       `json:"url"`
	CacheHit                 bool         `json:"cache_hit"`
	InputTokens              int          `json:"input_tokens"`
	OutputTokens             int          `json:"output_tokens"`
	TotalTokens              int          `json:"total_tokens"`
	LlmErrorMessage          string       `json:"llm_error_message"`
	ChatCompletionTimeTaken  int          `json:"llm_locatr_generation_time_taken"`
	AttemptNo                int          `json:"attempt_no"`
	LocatrRequestInitiatedAt time.Time    `json:"request_initiated_at"`
	LocatrRequestCompletedAt time.Time    `json:"request_completed_at"`
	AllLocatrs               []string     `json:"all_locatrs"`
	SelectorType             SelectorType `json:"selector_type"`
}

type LocatrOutput struct {
	SelectorType SelectorType `json:"selector_type"`
	Selectors    []string     `json:"selector"`
}

type cachedLocatrsDto struct {
	LocatrName   string       `json:"locatr_name"`
	Locatrs      []string     `json:"locatrs"`
	SelectorType SelectorType `json:"selector_type"`
}

type BaseLocatr struct {
	plugin        PluginInterface
	llmClient     llm.LlmClientInterface
	reRankClient  reranker.ReRankInterface
	options       BaseLocatrOptions
	cachedLocatrs map[string][]cachedLocatrsDto
	initialized   bool
	locatrResults []LocatrResult
}

// BaseLocatrOptions is a struct that holds all the options for the locatr package
type BaseLocatrOptions struct {
	// CachePath is the path to the cache file
	CachePath string
	// UseCache is a flag to enable/disable cache
	UseCache bool

	// LocatrResultsFilePath is the path to the file where the locatr results will be written
	// If not provided, the results will be written to DEFAULT_LOCATR_RESULTS_FILE
	ResultsFilePath string

	// LLmClient is the client to interact with LLM
	LlmClient llm.LlmClientInterface

	// ReRankClient is the client to interact with ReRank
	ReRankClient reranker.ReRankInterface
}

var (
	ErrUnableToMinifyHtmlDom            = errors.New("unable to minify HTML DOM")
	ErrUnableToExtractIdLocatorMap      = errors.New("unable to extract ID locator map")
	ErrUnableToLocateElementId          = errors.New("unable to locate element ID")
	ErrInvalidElementIdGenerated        = errors.New("invalid element ID generated")
	ErrUnableToFindValidLocator         = errors.New("unable to find valid locator")
	ErrFailedToWriteCache               = errors.New("failed to write cache")
	ErrFailedToMarshalJson              = errors.New("failed to marshal json")
	ErrLocatrCacheMiss                  = errors.New("cache miss")
	ErrLocatrRetrievalAttemptsExhausted = errors.New("failed to retrieve locatr after 3 attempts")
	ErrLocatrRetrievalFailed            = errors.New("failed to retieve locatr")
	ErrNoChunksToProcess                = errors.New("got no chunks to process after reranking")
	ErrFailedToRepariJson               = errors.New("failed to repair json")
)

// Default cache path
const DEFAULT_CACHE_PATH = ".locatr.cache"

// Default file to write locatr results
const DEFAULT_LOCATR_RESULTS_PATH = "locatr_results.json"

// CHUNK_SIZE is the maximum size of a html chunk
const CHUNK_SIZE = 4000

const MAX_RETRIES_WITH_RERANK = 3

const MAX_CHUNKS_EACH_RERANK_ITERATION = 4

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
	locatr := &BaseLocatr{
		plugin:        plugin,
		options:       options,
		cachedLocatrs: make(map[string][]cachedLocatrsDto),
		initialized:   false,
		locatrResults: []LocatrResult{},
		reRankClient:  options.ReRankClient,
	}
	if options.LlmClient == nil {
		client, err := llm.CreateLlmClientFromEnv()
		if err != nil {
			logger.Logger.Error(fmt.Sprintf("Failed to create LLM client: %v", err))
			return nil
		}
		locatr.llmClient = client
	} else {
		locatr.llmClient = options.LlmClient
	}
	if options.ReRankClient == nil {
		locatr.reRankClient = reranker.CreateCohereClientFromEnv()
	}
	return locatr
}

func (l *LocatrResult) MarshalJSON() ([]byte, error) {
	type Alias LocatrResult
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

// GetLocatorStr returns the locator string for the given user request
func (l *BaseLocatr) GetLocatorStr(ctx context.Context, userReq string) (*LocatrOutput, error) {
	ctx, span := tracing.StartSpan(ctx, "GetLocatrStr")
	defer span.End()

	span.AddEvent("initializing state")

	l.initializeState()
	logger.Logger.Info(fmt.Sprintf("Getting locator for user request: `%s`", userReq))

	currentUrl := l.plugin.GetCurrentContext(ctx)

	span.AddEvent("loading locatrs from cache")
	locatr, err := l.loadLocatrsFromCache(ctx, userReq)
	if err == nil {
		return locatr, nil
	}

	logger.Logger.Info("Cache miss, starting dom minification")
	span.AddEvent("minifying dom and generating locator map")

	minifiedDOM, locatorsMap, selectorType, err := l.plugin.GetMinifiedDomAndLocatorMap(ctx)
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Failed to minify DOM and extract ID locator map: %v", err))
		return nil, ErrUnableToMinifyHtmlDom
	}

	logger.Logger.Info("Extracting element ID using LLM")
	span.AddEvent("locating element using LLM")

	llmOutputs, err := l.locateElementId(ctx, minifiedDOM.ContentStr(), userReq)
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Failed to locate element ID: %v", err))
		if len(llmOutputs) > 0 {
			l.locatrResults = append(l.locatrResults,
				createLocatrResultFromOutput(
					userReq, currentUrl, []string{}, llmOutputs, "",
				)...,
			)
		}
		return nil, ErrUnableToLocateElementId
	}

	locators, ok := (*locatorsMap)[llmOutputs[len(llmOutputs)-1].LocatorID]

	slog.Debug("Locators generated", slog.Int("count", len(locators)))
	span.AddEvent("candidate locators generated")

	if !ok {
		logger.Logger.Error("Invalid element ID generated")
		return nil, ErrInvalidElementIdGenerated
	}

	span.SetAttributes(attribute.Int("candidate-locator-count", len(locators)))

	validLocator, err := l.getValidLocator(ctx, locators)

	slog.Debug("Valid locators found", slog.String("valid_locator", strings.Join(validLocator, "\n")))

	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Failed to find valid locator: %v", err))
		return nil, ErrUnableToFindValidLocator
	}
	locatrOutput := &LocatrOutput{
		selectorType,
		validLocator,
	}
	l.locatrResults = append(l.locatrResults,
		createLocatrResultFromOutput(
			userReq,
			currentUrl,
			locators,
			llmOutputs,
			selectorType,
		)...,
	)
	if l.options.UseCache {
		span.AddEvent("updating cache with new locators")

		logger.Logger.Info(fmt.Sprintf("Adding locatrs of `%s` to cache", userReq))
		logger.Logger.Debug(fmt.Sprintf("Adding Locatrs of `%s`: `%v` to cache", userReq, locators))
		l.addCachedLocatrs(currentUrl, userReq, locatrOutput)
		value, err := json.Marshal(l.cachedLocatrs)
		if err != nil {
			logger.Logger.Error(fmt.Sprintf("Failed to marshal cache: %v", err))
			return nil, fmt.Errorf("%w: %w", ErrFailedToMarshalJson, err)
		}
		if err = writeLocatorsToCache(l.options.CachePath, value); err != nil {
			logger.Logger.Error(fmt.Sprintf("Failed to write cache: %v", err))
			return nil, fmt.Errorf("%w: %w", ErrFailedToWriteCache, err)
		}
	}
	return locatrOutput, nil
}

func (l *BaseLocatr) getValidLocator(ctx context.Context, locators []string) ([]string, error) {
	locatrsToReturn := []string{}
	for _, locator := range locators {
		ok, err := l.plugin.IsValidLocator(ctx, locator)
		if ok {
			locatrsToReturn = append(locatrsToReturn, locator)
			logger.Logger.Debug(fmt.Sprintf("Valid locator found: `%s`", locator))
		} else {
			logger.Logger.Debug(fmt.Sprintf("error while checking for valid locatr: %s", err.Error()))
		}
	}
	if len(locatrsToReturn) == 0 {
		return nil, fmt.Errorf("%v %v", ErrUnableToFindValidLocator, errors.New("all locatrs exhausted"))
	}
	return locatrsToReturn, nil
}

func (l *BaseLocatr) getReRankedChunks(ctx context.Context, htmlDom string, userReq string) ([]string, error) {
	span := trace.SpanFromContext(ctx)

	span.AddEvent("splitting html using reranker")
	chunks := reranker.SplitHtml(htmlDom, HTML_SEPARATORS, CHUNK_SIZE)
	logger.Logger.Debug(fmt.Sprintf("SplitHtml resulted in %d chunks.", len(chunks)))
	request := reranker.ReRankRequest{
		Query:     userReq,
		Documents: chunks,
	}

	span.AddEvent("reranking HTML chunks")
	reRankResults, err := l.reRankClient.ReRank(ctx, request)
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Failed to re-rank chunks: %v", err))
		return nil, fmt.Errorf("failed to re-rank chunks: %v", err)
	}
	for _, result := range *reRankResults {
		logger.Logger.Debug(fmt.Sprintf("Re-rank result index: %d, score: %f", result.Index, result.Score))
	}
	return sortRerankChunks(chunks, *reRankResults), nil
}

func (l *BaseLocatr) llmGetElementId(ctx context.Context, htmlDom string, userReq string) (*llmLocatorOutputDto, error) {
	jsonData, err := json.Marshal(&llm.LlmWebInputDto{
		HtmlDom: htmlDom,
		UserReq: userReq,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal web input json: %w", err)
	}

	prompt := fmt.Sprintf("%s%s", LOCATR_PROMPT, string(jsonData))

	llmResponse, err := l.llmClient.ChatCompletion(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to get response from LLM: %w", err)
	}

	llmLocatorOutput := &llmLocatorOutputDto{
		completionResponse: *llmResponse,
	}

	logger.Logger.Debug(fmt.Sprintf("LLM response: %s", llmResponse.Completion))

	llmResponse.Completion = fixLLmJson(llmResponse.Completion)

	logger.Logger.Debug(fmt.Sprintf("Repaired LLM response: %s", llmResponse.Completion))

	if err = json.Unmarshal([]byte(llmResponse.Completion), llmLocatorOutput); err != nil {
		return nil, fmt.Errorf(
			"failed to unmarshal llmLocatorOutputDto json: %w, expectd json: recevied: %s",
			err,
			llmResponse.Completion,
		)
	}
	return llmLocatorOutput, nil
}
func (l *BaseLocatr) getLocatrOutput(ctx context.Context, htmlDOM string, userReq string) (*locatrOutputDto, error) {
	result, err := l.llmGetElementId(ctx, htmlDOM, userReq)
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

func (l *BaseLocatr) locateElementId(ctx context.Context, htmlDOM string, userReq string) ([]locatrOutputDto, error) {
	ctx, span := tracing.StartSpan(ctx, "locateElementId")
	defer span.End()

	llmOutputs := []locatrOutputDto{}
	requestInitiatedAt := time.Now()
	if l.reRankClient == nil {
		span.AddEvent("sending full dom to llm")

		logger.Logger.Debug("No rerank client setup sending full dom to llm.")
		result, err := l.getLocatrOutput(ctx, htmlDOM, userReq)
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

	span.AddEvent("fetching reranked chunks")
	chunks, err := l.getReRankedChunks(ctx, htmlDOM, userReq)
	if err != nil {
		return llmOutputs, err
	}
	if len(chunks) == 0 {
		logger.Logger.Debug("No chunks to process")
		return llmOutputs, ErrNoChunksToProcess
	}

	span.SetAttributes(attribute.Int("chunk-count", len(chunks)))

	if len(chunks) == 1 {
		logger.Logger.Debug("Only one chunk to process, sending to llm.")
		result, err := l.getLocatrOutput(ctx, htmlDOM, userReq)
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

		span.AddEvent(fmt.Sprintf("attempt %d chunk", attempt))

		switch attempt {
		case 0:
			domToProcess = strings.Join(chunks[0:MAX_RETRIES_WITH_RERANK], "\n")
		case 1:
			endIndex := MAX_CHUNKS_EACH_RERANK_ITERATION * 2
			if endIndex > len(chunks) {
				endIndex = len(chunks)
				attempt++
				logger.Logger.Debug(fmt.Sprintf("Max chunks reached in attempt %d, this will be the final attempt.", attempt+1))
			}
			domToProcess = strings.
				Join(chunks[MAX_CHUNKS_EACH_RERANK_ITERATION:endIndex], "\n")
		default:
			domToProcess = htmlDOM
		}

		logger.Logger.Debug(fmt.Sprintf("attempt no (%d) to find locatr with reranking", attempt+1))
		requestCompletedAt := time.Now()

		result, err := l.llmGetElementId(ctx, domToProcess, userReq)
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

		logger.Logger.Error(fmt.Sprintf("Failed to get locatr in %d attempt(s) : %s", attempt+1, result.Error))
	}
	return llmOutputs, ErrLocatrRetrievalAttemptsExhausted
}

func (l *BaseLocatr) WriteLocatrResultsToFile() {
	logger.Logger.Info(fmt.Sprintf("Writing locatr results to file: %s", l.options.ResultsFilePath))
	file, err := os.OpenFile(l.options.ResultsFilePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Failed to create file locatr results file: %v", err))
		return
	}
	defer file.Close()
	logger.Logger.Debug(fmt.Sprintf("Results to write: %v", l.locatrResults))
	value, err := json.Marshal(l.locatrResults)
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Failed to marshal locatr results json: %v", err))
		return
	}
	if _, err := file.Write(value); err != nil {
		logger.Logger.Error(fmt.Sprintf("Failed to write locatr results to file: %v", err))
	} else {
		logger.Logger.Info(fmt.Sprintf("Results written to file: %s", l.options.ResultsFilePath))
	}
}

func (l *BaseLocatr) GetLocatrResults() []LocatrResult {
	return l.locatrResults
}

func (l *BaseLocatr) addCachedLocatrs(
	url string,
	locatrName string,
	locatrOutput *LocatrOutput,
) {
	if _, ok := l.cachedLocatrs[url]; !ok {
		logger.Logger.Debug(fmt.Sprintf("Domain `%s` not found in cache... Creating new cachedLocatrsDto", url))
		l.cachedLocatrs[url] = []cachedLocatrsDto{}
	}
	found := false
	for i, v := range l.cachedLocatrs[url] {
		if v.LocatrName == locatrName {
			logger.Logger.Debug(fmt.Sprintf("Found locatr `%s` in cache... Updating locators", locatrName))
			l.cachedLocatrs[url][i].Locatrs =
				getUniqueStringArray(
					append(l.cachedLocatrs[url][i].Locatrs,
						locatrOutput.Selectors...,
					))
			l.cachedLocatrs[url][i].SelectorType = locatrOutput.SelectorType
			return
		}
	}
	if !found {
		logger.Logger.Debug(fmt.Sprintf("Locatr `%s` not found in cache... Creating new locatr", locatrName))
		l.cachedLocatrs[url] =
			append(l.cachedLocatrs[url],
				cachedLocatrsDto{
					LocatrName:   locatrName,
					Locatrs:      locatrOutput.Selectors,
					SelectorType: locatrOutput.SelectorType,
				})
	}
}
func (l *BaseLocatr) getLocatrsFromState(key string, currentContext string) ([]string, SelectorType, error) {
	if locatrs, ok := l.cachedLocatrs[currentContext]; ok {
		for _, v := range locatrs {
			if v.LocatrName == key {
				logger.Logger.Debug(fmt.Sprintf("Key `%s` found in cache", key))
				return v.Locatrs, v.SelectorType, nil
			}
		}
	}
	logger.Logger.Debug(fmt.Sprintf("Key `%s not found in cache", key))
	return nil, "", fmt.Errorf("key `%s` not found in cache", key)
}

func (l *BaseLocatr) loadLocatrsFromCache(ctx context.Context, userReq string) (*LocatrOutput, error) {
	ctx, span := tracing.StartSpan(ctx, "loadLocatorsFromCache")
	defer span.End()

	requestInitiatedAt := time.Now()
	currentContext := l.plugin.GetCurrentContext(ctx)

	span.AddEvent("fetching locators from cache")
	locators, selectorType, err := l.getLocatrsFromState(userReq, currentContext)

	if err != nil {
		span.AddEvent("failed to load locators from cache")

		logger.Logger.Error(fmt.Sprintf("Failed to get locators from cache: %v", err))
		return nil, err

	} else {
		if len(locators) > 0 {
			span.AddEvent("validating locators from cache")
			span.SetAttributes(attribute.Int("candidate-locators", len(locators)))

			validLocators, err := l.getValidLocator(ctx, locators)
			if err == nil {
				result := LocatrResult{
					LocatrDescription:        userReq,
					CacheHit:                 true,
					AllLocatrs:               validLocators,
					Url:                      currentContext,
					LocatrRequestInitiatedAt: requestInitiatedAt,
					LocatrRequestCompletedAt: time.Now(),
					SelectorType:             selectorType,
				}
				l.locatrResults = append(l.locatrResults, result)
				logger.Logger.Info(fmt.Sprintf("Cache hit, key: `%s`, value: `%s`", userReq, validLocators))
				return &LocatrOutput{
					SelectorType: (selectorType),
					Selectors:    locators,
				}, nil
			} else {
				span.AddEvent("valid locator not found in cache")
				logger.Logger.Error(fmt.Sprintf("Failed to find valid locator in cache: %v", err))
			}
			logger.Logger.Info("All cached locators are outdated.")
		}

	}
	return nil, ErrLocatrCacheMiss
}

func (l *BaseLocatr) loadLocatorsCache(cachePath string) error {
	file, err := os.Open(cachePath)
	if err != nil {
		logger.Logger.Debug(fmt.Sprintf("Cache file not found: %v", err))
		return nil // ignore this error for now
	}
	defer file.Close()
	byteValue, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read cache file `(%s)`: %v", cachePath, err)
	}
	err = json.Unmarshal(byteValue, &l.cachedLocatrs)
	if err != nil {
		return fmt.Errorf(
			"failed to unmarshal cache file `(%s)`: %v, expected json received: %s",
			cachePath,
			err,
			byteValue,
		)
	}
	return nil
}
func writeLocatorsToCache(cachePath string, cacheString []byte) error {
	err := os.MkdirAll(filepath.Dir(cachePath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	file, err := os.OpenFile(cachePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()
	if _, err := file.Write(cacheString); err != nil {
		return fmt.Errorf("failed to write cache: %v", err)
	}

	return nil
}
func (l *BaseLocatr) initializeState() {
	if l.initialized || !l.options.UseCache {
		logger.Logger.Debug("Cache disabled or already initialized")
		return
	}
	err := l.loadLocatorsCache(l.options.CachePath)
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Failed to load cache: %v", err))
		return
	}
	logger.Logger.Debug("Cache loaded successfully")
	l.initialized = true
}

func getUniqueStringArray(input []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range input {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func createLocatrResultFromOutput(
	userReq string,
	currentUrl string,
	allLocatrs []string,
	output []locatrOutputDto,
	selectorType SelectorType,
) []LocatrResult {
	results := []LocatrResult{}
	for _, outputDto := range output {
		r := LocatrResult{
			LocatrDescription:        userReq,
			CacheHit:                 false,
			InputTokens:              outputDto.completionResponse.InputTokens,
			OutputTokens:             outputDto.completionResponse.OutputTokens,
			TotalTokens:              outputDto.completionResponse.TotalTokens,
			ChatCompletionTimeTaken:  outputDto.completionResponse.TimeTaken,
			Url:                      currentUrl,
			LocatrRequestInitiatedAt: outputDto.LocatrRequestInitiatedAt,
			LocatrRequestCompletedAt: outputDto.LocatrRequestCompletedAt,
			AttemptNo:                outputDto.AttemptNo,
			LlmErrorMessage:          outputDto.Error,
			AllLocatrs:               allLocatrs,
			SelectorType:             selectorType,
		}
		results = append(results, r)

	}
	return results
}

func sortRerankChunks(chunks []string, reRankResults []reranker.ReRankResult) []string {
	// Filter out results with indices out of range
	validResults := []reranker.ReRankResult{}
	for _, result := range reRankResults {
		if result.Index < len(chunks) {
			validResults = append(validResults, result)
		}
	}

	// If no valid results, return the original chunks
	if len(validResults) == 0 {
		return chunks
	}

	// Sort chunks based on valid rerank results
	finalChunks := []string{}
	for _, result := range validResults {
		finalChunks = append(finalChunks, chunks[result.Index])
	}
	return finalChunks
}

func fixLLmJson(json string) string {
	json = strings.TrimPrefix(json, "```")
	json = strings.TrimPrefix(json, "json")
	json = strings.TrimSuffix(json, "```")

	return json
}
