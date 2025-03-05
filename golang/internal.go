package locatr

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/vertexcover-io/locatr/golang/constants"
	"github.com/vertexcover-io/locatr/golang/reranker/splitters"
	"github.com/vertexcover-io/locatr/golang/types"
)

// loadCache reads and deserializes the cache file into memory.
// Returns nil if the cache file doesn't exist, error if reading or parsing fails.
func (l *Locatr) loadCache() error {

	file, err := os.Open(l.options.CachePath)
	if err != nil {
		// return nil if file doesn't exist. It will be created later by persistCache method
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("couldn't open file: %v", err)
	}
	defer file.Close()

	byteContent, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("couldn't read file: %v", err)
	}
	if len(byteContent) == 0 {
		return nil
	}

	if err := json.Unmarshal(byteContent, &l.cache); err != nil {
		return fmt.Errorf("error loading cache: %v", err)
	}
	return nil
}

// persistCache serializes and writes the current cache to disk.
// Creates necessary directories and handles file permissions.
// Returns error if writing fails.
func (l *Locatr) persistCache() error {
	cacheBytes, err := json.Marshal(l.cache)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(l.options.CachePath), 0755); err != nil {
		return err
	}
	file, err := os.OpenFile(l.options.CachePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	if _, err := file.Write(cacheBytes); err != nil {
		return fmt.Errorf("failed to write cache: %v", err)
	}
	return nil
}

// processCacheRequest attempts to find matching locators in the cache.
// Parameters:
//   - completion: Output structure to populate with cache results
//   - userRequest: Natural language description to look up
//
// Returns error if no valid cached locators are found.
// Includes retry logic for elements that may need time to appear in DOM.
func (l *Locatr) processCacheRequest(completion *types.LocatrCompletion, userRequest string) error {
	if entries, ok := l.cache[l.plugin.GetCurrentContext()]; ok {
		retry := false

	retryLoop: // TODO: Document why we are taking this approach
		for _, entry := range entries {
			if entry.UserRequest != userRequest {
				continue
			}

			validLocators := []string{}
			for i, locator := range entry.Locators {
				ok, err := l.plugin.IsLocatorValid(locator)
				if err != nil || !ok {
					if i == 0 && !retry {
						log.Println("Elements may not be available yet, retrying in 2 seconds")
						retry = true
						time.Sleep(2 * time.Second)
						goto retryLoop
					}
					continue
				}
				validLocators = append(validLocators, locator)
			}

			if len(validLocators) > 0 {
				log.Printf("Cache hit: %v\n", entry.UserRequest)
				completion.Locators = validLocators
				completion.LocatorType = entry.LocatorType
				completion.CacheHit = true
				return nil
			}
		}
	}
	return fmt.Errorf("no cache entry found for user request: %v", userRequest)
}

// processIdRequest finds elements by analyzing DOM structure and element IDs.
// Parameters:
//   - completion: Output structure to populate with results
//   - userRequest: Natural language description of target element
//   - dom: Current page DOM representation
//
// Returns error if no matching elements are found.
// Updates completion with token usage and timing metrics.
func (l *Locatr) processIdRequest(completion *types.LocatrCompletion, userRequest string, dom *types.DOM) error {
	domChunks, err := l.getRerankedChunks(dom.RootElement.Repr(), userRequest)
	if err != nil {
		return err
	}
	if len(domChunks) == 0 {
		return fmt.Errorf("no chunks to process")
	}

	locatorMap := dom.Metadata.LocatorMap
	// TODO: Run evals to verify If this approach needs to be changed
	for _, chunk := range domChunks {
		idComletion, err := l.options.LLMClient.GetIdCompletion(userRequest, chunk)
		completion.TimeTaken += idComletion.TimeTaken
		completion.InputTokens += idComletion.InputTokens
		completion.OutputTokens += idComletion.OutputTokens
		completion.TotalTokens += idComletion.TotalTokens
		if err != nil {
			log.Printf("couldn't get ID completion: %v\n", err)
			continue
		}
		if idComletion.ErrorMessage != "" {
			log.Printf("error getting relevant ID: %v\n", idComletion.ErrorMessage)
			continue
		}
		locators := locatorMap[idComletion.Id]
		if len(locators) == 0 {
			log.Printf("no locators found for ID: %v\n", idComletion.Id)
			continue
		}
		completion.Locators = locators
		completion.LocatorType = dom.Metadata.LocatorType
		return nil
	}
	return errors.New("no relevant locator ID found in the DOM")
}

// processPointRequest finds elements using visual grounding via screenshots.
// Parameters:
//   - completion: Output structure to populate with results
//   - userRequest: Natural language description of target element
//   - dom: Current page DOM representation
//
// Returns error if no matching elements are found.
// Process includes:
// 1. Setting viewport size
// 2. Scrolling to candidate elements
// 3. Taking screenshots
// 4. Using LLM to identify click points
// 5. Converting points to locators
func (l *Locatr) processPointRequest(completion *types.LocatrCompletion, userRequest string, dom *types.DOM) error {
	domChunks, err := l.getRerankedChunks(dom.RootElement.Repr(), userRequest)
	if err != nil {
		return err
	}
	if len(domChunks) == 0 {
		return fmt.Errorf("no chunks to process")
	}

	locatorMap := dom.Metadata.LocatorMap
	for _, chunk := range domChunks {
		id, err := extractFirstUniqueID(chunk)
		if err != nil {
			continue
		}
		l.plugin.SetViewportSize(1280, 800)

		locator := locatorMap[id][0]
		scrollPosition, err := l.plugin.ScrollToLocator(locator)
		if err != nil {
			log.Printf("couldn't scroll to locator: %v\n", err)
			continue
		}
		screenshot, err := l.plugin.TakeScreenshot()
		if err != nil {
			log.Printf("couldn't take screenshot: %v\n", err)
			continue
		}
		pointCompletion, err := l.options.LLMClient.GetPointCompletion(userRequest, screenshot)
		completion.TimeTaken += pointCompletion.TimeTaken
		completion.InputTokens += pointCompletion.InputTokens
		completion.OutputTokens += pointCompletion.OutputTokens
		completion.TotalTokens += pointCompletion.TotalTokens
		if err != nil {
			log.Printf("couldn't get point completion: %v\n", err)
			continue
		}
		if pointCompletion.ErrorMessage != "" {
			log.Printf("error getting relevant point: %v\n", pointCompletion.ErrorMessage)
			continue
		}
		locators, err := l.plugin.GetLocatorsFromPoint(&pointCompletion.Point, scrollPosition)
		if err != nil {
			log.Println(err)
			continue
		}
		if len(locators) == 0 {
			log.Printf("no locators found at point: %v\n", pointCompletion.Point)
			continue
		}
		completion.Locators = locators
		completion.LocatorType = dom.Metadata.LocatorType
		return nil
	}
	return errors.New("no relevant point found in the DOM")
}

// getRerankedChunks splits DOM into chunks and ranks them by relevance to the user request.
// Parameters:
//   - dom: HTML string representing the current page
//   - userRequest: Natural language description for ranking relevance
//
// Returns sorted chunks based on relevance scores and any error that occurred.
func (l *Locatr) getRerankedChunks(dom string, userRequest string) ([]string, error) {
	chunks := splitters.SplitHtml(dom, constants.HTML_SEPARATORS, constants.CHUNK_SIZE)
	log.Printf("Number of chunks to process: %d\n", len(chunks))

	results, err := l.options.ReRanker.ReRank(
		&types.ReRankRequest{Query: userRequest, Documents: chunks},
	)
	if err != nil {
		return nil, err
	}
	return sortRerankChunks(chunks, results), nil
}
