package baseLocatr

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	locatr "github.com/vertexcover-io/locatr/golang"
	"github.com/vertexcover-io/locatr/golang/elementSpec"
	"github.com/vertexcover-io/locatr/golang/llm"
	"github.com/vertexcover-io/locatr/golang/logger"
	"github.com/vertexcover-io/locatr/golang/reranker"
)

type PluginInterface interface {
	GetMinifiedDomAndLocatorMap() (
		*elementSpec.ElementSpec,
		*elementSpec.IdToLocatorMap,
		error,
	)
	GetCurrentContext() string
	IsValidLocator(locatr string) (string, error)
}

type LocatrInterface interface {
	WriteResultsToFile()
	GetLocatrResults() []LocatrResult
	GetLocatrStr(userReq string) (string, error)
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

type cachedLocatrsDto struct {
	LocatrName string   `json:"locatr_name"`
	Locatrs    []string `json:"locatrs"`
}

type BaseLocatr struct {
	plugin        PluginInterface
	llmClient     llm.LlmClientInterface
	reRankClient  reranker.ReRankInterface
	options       BaseLocatrOptions
	cachedLocatrs map[string][]cachedLocatrsDto
	initialized   bool
	logger        logger.LogInterface
	locatrResults []LocatrResult
}

// BaseLocatrOptions is a struct that holds all the options for the locatr package
type BaseLocatrOptions struct {
	// CachePath is the path to the cache file
	CachePath string
	// UseCache is a flag to enable/disable cache
	UseCache bool
	// LogConfig is the log configuration
	LogConfig logger.LogConfig

	// LocatrResultsFilePath is the path to the file where the locatr results will be written
	// If not provided, the results will be written to DEFAULT_LOCATR_RESULTS_FILE
	ResultsFilePath string

	// LLmClient is the client to interact with LLM
	LlmClient llm.LlmClientInterface

	// ReRankClient is the client to interact with ReRank
	ReRankClient reranker.ReRankInterface
}

var (
	ErrUnableToMinifyHtmlDom       = errors.New("unable to minify HTML DOM")
	ErrUnableToExtractIdLocatorMap = errors.New("unable to extract ID locator map")
	ErrUnableToLocateElementId     = errors.New("unable to locate element ID")
	ErrInvalidElementIdGenerated   = errors.New("invalid element ID generated")
	ErrUnableToFindValidLocator    = errors.New("unable to find valid locator")
	ErrFailedToWriteCache          = errors.New("failed to write cache")
	ErrFailedToMarshalJson         = errors.New("failed to marshal json")
)

var ErrLocatrCacheMiss = errors.New("cache miss")

var ErrLocatrRetrievalAttemptsExhausted = errors.New("failed to retrieve locatr after 3 attempts")

var ErrLocatrRetrievalFailed = errors.New("failed to retieve locatr")

var ErrNoChunksToProcess = errors.New("got no chunks to process after reranking")

var ErrFailedToRepariJson = errors.New("failed to repair json")

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
	if options.LogConfig.Writer == nil {
		options.LogConfig.Writer = logger.DefaultLogWriter
	}
	locatr := &BaseLocatr{
		plugin:        plugin,
		options:       options,
		cachedLocatrs: make(map[string][]cachedLocatrsDto),
		initialized:   false,
		logger:        logger.NewLogger(options.LogConfig),
		locatrResults: []LocatrResult{},
		reRankClient:  options.ReRankClient,
	}
	if options.LlmClient == nil {
		client, err := llm.CreateLlmClientFromEnv()
		if err != nil {
			locatr.logger.Error(fmt.Sprintf("Failed to create LLM client: %v", err))
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
func (l *BaseLocatr) GetLocatorStr(userReq string) (string, error) {
	l.initializeState()
	l.logger.Info(fmt.Sprintf("Getting locator for user request: `%s`", userReq))
	currentUrl := l.plugin.GetCurrentContext()
	locatr, err := l.loadLocatrsFromCache(userReq)
	if err == nil {
		return locatr, nil
	}
	l.logger.Info("Cache miss, starting dom minification")
	minifiedDOM, locatorsMap, err := l.plugin.GetMinifiedDomAndLocatorMap()
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
					userReq, "", currentUrl, []string{}, llmOutputs,
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
			userReq,
			validLocator,
			currentUrl,
			locators,
			llmOutputs,
		)...,
	)
	if l.options.UseCache {
		l.logger.Info(fmt.Sprintf("Adding locatrs of `%s` to cache", userReq))
		l.logger.Debug(fmt.Sprintf("Adding Locatrs of `%s`: `%v` to cache", userReq, locators))
		l.addCachedLocatrs(currentUrl, userReq, locators)
		value, err := json.Marshal(l.cachedLocatrs)
		if err != nil {
			l.logger.Error(fmt.Sprintf("Failed to marshal cache: %v", err))
			return "", fmt.Errorf("%w: %w", ErrFailedToMarshalJson, err)
		}
		if err = writeLocatorsToCache(l.options.CachePath, value); err != nil {
			l.logger.Error(fmt.Sprintf("Failed to write cache: %v", err))
			return "", fmt.Errorf("%w: %w", ErrFailedToWriteCache, err)
		}
	}
	return validLocator, nil

}

func (l *BaseLocatr) getValidLocator(locators []string) (string, error) {
	for _, locator := range locators {
		value, err := l.plugin.IsValidLocator(locator)
		if value != "" {
			l.logger.Debug(fmt.Sprintf("Valid locator found: `%s`", locator))
			return locator, nil
		} else {
			l.logger.Debug(fmt.Sprintf("error while evaluating js function %v", err))
			return "", fmt.Errorf("%w %w", ErrUnableToFindValidLocator, err)
		}
	}
	return "", fmt.Errorf("%v %v", ErrUnableToFindValidLocator, errors.New("all locatrs exhausted"))
}
func (l *BaseLocatr) getReRankedChunks(htmlDom string, userReq string) ([]string, error) {
	chunks := reranker.SplitHtml(htmlDom, locatr.HTML_SEPARATORS, CHUNK_SIZE)
	l.logger.Debug(fmt.Sprintf("SplitHtml resulted in %d chunks.", len(chunks)))
	request := reranker.ReRankRequest{
		Query:     userReq,
		Documents: chunks,
	}
	reRankResults, err := l.reRankClient.ReRank(request)
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
	jsonData, err := json.Marshal(&llm.LlmWebInputDto{
		HtmlDom: htmlDom,
		UserReq: userReq,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal web input json: %w", err)
	}

	prompt := fmt.Sprintf("%s%s", locatr.LOCATR_PROMPT, string(jsonData))

	llmResponse, err := l.llmClient.ChatCompletion(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to get response from LLM: %w", err)
	}

	llmLocatorOutput := &llmLocatorOutputDto{
		completionResponse: *llmResponse,
	}

	l.logger.Debug(fmt.Sprintf("LLM response: %s", llmResponse.Completion))

	llmResponse.Completion = fixLLmJson(llmResponse.Completion)

	l.logger.Debug(fmt.Sprintf("Repaired LLM response: %s", llmResponse.Completion))

	if err = json.Unmarshal([]byte(llmResponse.Completion), llmLocatorOutput); err != nil {
		return nil, fmt.Errorf("failed to unmarshal llmLocatorOutputDto json: %w", err)
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

func (l *BaseLocatr) WriteLocatrResultsToFile() {
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

func (l *BaseLocatr) GetLocatrResults() []LocatrResult {
	return l.locatrResults
}

func (l *BaseLocatr) addCachedLocatrs(url string, locatrName string, locatrs []string) {
	if _, ok := l.cachedLocatrs[url]; !ok {
		l.logger.Debug(fmt.Sprintf("Domain `%s` not found in cache... Creating new cachedLocatrsDto", url))
		l.cachedLocatrs[url] = []cachedLocatrsDto{}
	}
	found := false
	for i, v := range l.cachedLocatrs[url] {
		if v.LocatrName == locatrName {
			l.logger.Debug(fmt.Sprintf("Found locatr `%s` in cache... Updating locators", locatrName))
			l.cachedLocatrs[url][i].Locatrs = getUniqueStringArray(append(l.cachedLocatrs[url][i].Locatrs, locatrs...))
			return
		}
	}
	if !found {
		l.logger.Debug(fmt.Sprintf("Locatr `%s` not found in cache... Creating new locatr", locatrName))
		l.cachedLocatrs[url] = append(l.cachedLocatrs[url], cachedLocatrsDto{LocatrName: locatrName, Locatrs: locatrs})
	}
}
func (l *BaseLocatr) getLocatrsFromState(key string, currentUrl string) ([]string, error) {
	if locatrs, ok := l.cachedLocatrs[currentUrl]; ok {
		for _, v := range locatrs {
			if v.LocatrName == key {
				l.logger.Debug(fmt.Sprintf("Key `%s` found in cache", key))
				return v.Locatrs, nil
			}
		}
	}
	l.logger.Debug(fmt.Sprintf("Key `%s not found in cache", key))
	return nil, fmt.Errorf("key `%s` not found in cache", key)
}
func (l *BaseLocatr) loadLocatrsFromCache(userReq string) (string, error) {
	requestInitiatedAt := time.Now()
	currentUrl := l.plugin.GetCurrentContext()
	locators, err := l.getLocatrsFromState(userReq, currentUrl)

	if err != nil {
		l.logger.Error(fmt.Sprintf("Failed to get locators from cache: %v", err))
		return "", err
	} else {
		if len(locators) > 0 {
			validLocator, err := l.getValidLocator(locators)
			if err == nil {
				result := LocatrResult{
					LocatrDescription:        userReq,
					CacheHit:                 true,
					Locatr:                   validLocator,
					Url:                      currentUrl,
					LocatrRequestInitiatedAt: requestInitiatedAt,
					LocatrRequestCompletedAt: time.Now(),
				}
				l.locatrResults = append(l.locatrResults, result)
				l.logger.Info(fmt.Sprintf("Cache hit, key: `%s`, value: `%s`", userReq, validLocator))
				return validLocator, nil
			} else {
				l.logger.Error(fmt.Sprintf("Failed to find valid locator in cache: %v", err))
			}
			l.logger.Info("All cached locators are outdated.")
		}

	}
	return "", ErrLocatrCacheMiss
}

func (l *BaseLocatr) loadLocatorsCache(cachePath string) error {
	file, err := os.Open(cachePath)
	if err != nil {
		l.logger.Debug(fmt.Sprintf("Cache file not found: %v", err))
		return nil // ignore this error for now
	}
	defer file.Close()
	byteValue, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read cache file `(%s)`: %v", cachePath, err)
	}
	err = json.Unmarshal(byteValue, &l.cachedLocatrs)
	if err != nil {
		return fmt.Errorf("failed to unmarshal cache file `(%s)`: %v", cachePath, err)
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
		l.logger.Debug("Cache disabled or already initialized")
		return
	}
	err := l.loadLocatorsCache(l.options.CachePath)
	if err != nil {
		l.logger.Error(fmt.Sprintf("Failed to load cache: %v", err))
		return
	}
	l.logger.Debug("Cache loaded successfully")
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
	userReq string, validLocatr string,
	currentUrl string, allLocatrs []string,
	output []locatrOutputDto) []LocatrResult {
	results := []LocatrResult{}
	for _, outputDto := range output {
		r := LocatrResult{
			LocatrDescription:        userReq,
			CacheHit:                 false,
			Locatr:                   validLocatr,
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
		}
		results = append(results, r)

	}
	return results
}

func sortRerankChunks(chunks []string, reRankResults []reranker.ReRankResult) []string {
	finalChunks := []string{}
	for _, result := range reRankResults {
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
