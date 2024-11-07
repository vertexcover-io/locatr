package locatr

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
)

//go:embed meta/htmlMinifier.js
var HTML_MINIFIER_JS_CONTENTT string

type llmLocatorOutputDto struct {
	LocatorID          string `json:"locator_id"`
	completionResponse chatCompletionResponse
}

type locatrResult struct {
	LocatrDescription       string `json:"locatr_description"`
	CacheHit                bool   `json:"cache_hit"`
	Locatr                  string `json:"locatr"`
	InputTokens             int    `json:"input_tokens"`
	OutputTokens            int    `json:"output_tokens"`
	TotalTokens             int    `json:"total_tokens"`
	ChatCompletionTimeTaken int    `json:"llm_locatr_generation_time_taken"`
}

type cachedLocatrsDto struct {
	LocatrName string   `json:"locatr_name"`
	Locatrs    []string `json:"locatrs"`
}

type BaseLocatr struct {
	plugin        PluginInterface
	llmClient     LlmClientInterface
	reRankClinet  ReRankInterface
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
		reRankClinet:  options.ReRankClient,
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
	if err := l.plugin.evaluateJsScript(HTML_MINIFIER_JS_CONTENTT); err != nil {
		return "", ErrUnableToLoadJsScripts
	}
	l.initializeState()
	l.logger.Info(fmt.Sprintf("Getting locator for user request: %s", userReq))
	currentUrl := l.getCurrentUrl()
	locatr, err := l.loadLocatrsFromCache(userReq)
	if err == nil {
		return locatr, nil
	} else {
		l.logger.Error(fmt.Sprintf("Failed to load locatrs from cache: %v", err))
	}

	l.logger.Info("Cache miss, starting dom minification")
	minifiedDOM, locatorsMap, err := l.getMinifiedDomAndLocatorMap()
	if err != nil {
		l.logger.Error(fmt.Sprintf("Failed to minify DOM and extract ID locator map: %v", err))
		return "", ErrUnableToMinifyHtmlDom
	}

	l.logger.Info("Extracting element ID using LLM")
	llmOtuput, err := l.locateElementId(minifiedDOM.ContentStr(), userReq)
	// file, err := os.Create("temp.html")
	// if err != nil {
	// 	l.logger.Error(fmt.Sprintf("Failed to create temp file: %v", err))
	// }
	// defer file.Close()
	// file.WriteString(minifiedDOM.ContentStr())
	// l.logger.Info("Temp file created")
	if err != nil {
		l.logger.Error(fmt.Sprintf("Failed to locate element ID: %v", err))
		return "", ErrUnableToLocateElementId
	}

	locators, ok := (*locatorsMap)[llmOtuput.LocatorID]
	if !ok {
		l.logger.Error("Invalid element ID generated")
		return "", ErrInvalidElementIdGenerated
	}

	validLocator, err := l.getValidLocator(locators)
	if err != nil {
		l.logger.Error(fmt.Sprintf("Failed to find valid locator: %v", err))
		return "", ErrUnableToFindValidLocator
	}
	if l.options.UseCache {
		l.logger.Info(fmt.Sprintf("Adding locatrs of %s to cache", userReq))
		l.logger.Debug(fmt.Sprintf("Adding Locars of %s: %v to cache", userReq, locators))
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
	result := locatrResult{
		LocatrDescription:       userReq,
		CacheHit:                false,
		Locatr:                  validLocator,
		InputTokens:             llmOtuput.completionResponse.InputTokens,
		OutputTokens:            llmOtuput.completionResponse.OutputTokens,
		TotalTokens:             llmOtuput.completionResponse.TotalTokens,
		ChatCompletionTimeTaken: llmOtuput.completionResponse.TimeTaken,
	}

	l.locatrResults = append(l.locatrResults, result)
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
			return locator, nil
		}
	}
	return "", ErrUnableToFindValidLocator
}
func (l *BaseLocatr) getReRankedChunks(htmlDom string, userReq string) ([]string, error) {
	chunks := splitHtml(htmlDom, ChunkSize)
	request := reRankRequest{
		Query:     userReq,
		Documents: chunks,
	}
	reRankResults, err := l.reRankClinet.reRank(request)
	if err != nil {
		return nil, fmt.Errorf("failed to re-rank chunks: %v", err)
	}
	finalChunks := []string{}
	for _, result := range *reRankResults {
		if result.Score >= CohereReRankThreshold {
			finalChunks = append(finalChunks, chunks[result.Index])
		}
	}
	return finalChunks, nil

}
func (l *BaseLocatr) locateElementId(htmlDOM string, userReq string) (*llmLocatorOutputDto, error) {
	if l.reRankClinet != nil {
		chunks, err := l.getReRankedChunks(htmlDOM, userReq)
		if err != nil {
			l.logger.Error(fmt.Sprintf("Failed to re-rank chunks: %v", err))
			return nil, err
		}
		htmlDOM = ""
		for _, chunk := range chunks {
			htmlDOM += chunk
		}
	}
	fmt.Println("HTML DOM", htmlDOM)
	jsonData, err := json.Marshal(&llmWebInputDto{
		HtmlDom: htmlDOM,
		UserReq: userReq,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal web input json: %v", err)
	}

	prompt := fmt.Sprintf("%s%s", locatrPrompt, string(jsonData))

	llmResponse, err := l.llmClient.ChatCompletion(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to get response from LLM: %v", err)
	}

	llmLocatorOutput := &llmLocatorOutputDto{
		completionResponse: *llmResponse,
	}
	if err = json.Unmarshal([]byte(llmResponse.Completion), llmLocatorOutput); err != nil {
		return nil, fmt.Errorf("failed to unmarshal llmLocatorOutputDto json: %v", err)
	}

	return llmLocatorOutput, nil
}

func (l *BaseLocatr) writeLocatrResultsToFile() {
	file, err := os.OpenFile(l.options.ResultsFilePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		l.logger.Error(fmt.Sprintf("Failed to create file locatr results file: %v", err))
		return
	}
	defer file.Close()
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
