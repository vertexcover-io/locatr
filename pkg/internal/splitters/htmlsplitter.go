// Borrowed from: https://github.com/langchain-ai/langchain/blob/master/libs/text-splitters/langchain_text_splitters/character.py
// File contains the implementation of RecursiveCharacterTextSplitter

// Package splitters provides text chunking strategies optimized for different content types.
// This implementation is adapted from LangChain's RecursiveCharacterTextSplitter.
package splitters

import (
	"log"
	"regexp"
	"strings"

	"github.com/vertexcover-io/locatr/pkg/internal/constants"
)

// splitWithSeparator splits text into chunks using a regular expression separator.
// Parameters:
//   - text: The input text to split
//   - separator: Regular expression pattern to use as split points
//
// Returns an array of strings containing:
//   - The text before each separator
//   - The separator itself
//   - The remaining text after the last separator
//
// Note: This preserves the separators in the output, unlike strings.Split()
func splitWithSeparator(text, separator string) []string {
	re := regexp.MustCompile(separator)

	var result []string
	lastIndex := 0
	for _, indices := range re.FindAllStringIndex(text, -1) {
		result = append(result, text[lastIndex:indices[0]])  // text part before separator
		result = append(result, text[indices[0]:indices[1]]) // the separator itself
		lastIndex = indices[1]
	}
	result = append(result, text[lastIndex:]) // remaining text after last separator

	return result
}

// splitTextWithRegex splits text using a regular expression pattern.
// Parameters:
//   - text: The input text to split
//   - separator: Regular expression pattern to use as split points
//   - keepSeparator: If true, includes separators in the output chunks
//
// Returns an array of non-empty text chunks.
func splitTextWithRegex(
	text string, separator string, keepSeparator bool,
) []string {
	splits := []string{}
	if separator != "" {
		if keepSeparator {
			_splits := splitWithSeparator(text, separator)
			// Combine separators with following text
			for i := 1; i < len(_splits); i += 2 {
				if i < len(_splits)-1 {
					splits = append(splits, _splits[i]+_splits[i+1])
				} else {
					splits = append(splits, _splits[i])
				}
			}
			if len(_splits)%2 == 0 {
				splits = append(splits, _splits[len(_splits)-1])
			}
			// Prepend the first part of the text
			splits = append([]string{_splits[0]}, splits...)
		} else {
			splits = splitWithSeparator(text, separator)
		}
	} else {
		splits = []string{text}
	}

	// Remove empty chunks
	finalSplits := []string{}
	for _, split := range splits {
		if split != "" {
			finalSplits = append(finalSplits, split)
		}
	}
	return finalSplits
}

// joinDocs combines text chunks with a separator.
// Parameters:
//   - docs: Array of text chunks to join
//   - separator: String to insert between chunks
//   - stripWhitespace: If true, removes leading/trailing whitespace
//
// Returns pointer to joined string, or nil if result would be empty.
func joinDocs(docs []string, separator string, stripWhitespace bool) *string {
	text := strings.Join(docs, separator)
	if stripWhitespace {
		text = strings.TrimSpace(text)
	}
	if text == "" {
		return nil
	}
	return &text
}

// mergeSplits combines text chunks while respecting size constraints.
// Parameters:
//   - splits: Array of text chunks to merge
//   - separator: String to insert between merged chunks
//   - maxChunkSize: Maximum allowed size for merged chunks
//
// Returns array of merged chunks that don't exceed maxChunkSize.
// Uses a sliding window approach to combine chunks while maintaining overlap.
func mergeSplits(splits []string, separator string, maxChunkSize int) []string {
	separatorLength := len(separator)
	mergedDocs := []string{}
	currentChunks := []string{}
	currentSize := 0

	for _, split := range splits {
		splitLength := len(split)
		var additionalSeparatorLength int
		if len(currentChunks) > 0 {
			additionalSeparatorLength = separatorLength
		} else {
			additionalSeparatorLength = 0
		}

		if currentSize+splitLength+additionalSeparatorLength > maxChunkSize {
			if currentSize > maxChunkSize {
				log.Printf("Created chunk size of %d is greater than %d", currentSize, maxChunkSize)
			}
			if len(currentChunks) > 0 {
				mergedDoc := joinDocs(currentChunks, separator, false)
				if mergedDoc != nil {
					mergedDocs = append(mergedDocs, *mergedDoc)
				}

				// Remove chunks from the window until we can add the new split
				for {
					if len(currentChunks) > 0 {
						additionalSeparatorLength = separatorLength
					} else {
						additionalSeparatorLength = 0
					}
					if !((currentSize > constants.DEFAULT_CHUNK_OVERLAP) ||
						(currentSize+splitLength+additionalSeparatorLength > maxChunkSize) && (currentSize > 0)) {
						break
					}
					sizeReduction := 0
					if len(currentChunks) > 0 {
						sizeReduction = separatorLength
					}
					currentSize -= len(currentChunks[0]) + sizeReduction
					currentChunks = currentChunks[1:]
				}
			}
		}

		currentChunks = append(currentChunks, split)
		currentSize += splitLength
		if len(currentChunks) > 0 {
			currentSize += separatorLength
		}
	}
	mergedDoc := joinDocs(currentChunks, separator, false)
	if mergedDoc != nil {
		mergedDocs = append(mergedDocs, *mergedDoc)
	}
	return mergedDocs
}

// SplitHtml splits HTML content into chunks while preserving structure.
// Parameters:
//   - text: HTML content to split
//   - separators: Array of regex patterns to use as split points, in order of preference
//   - chunkSize: Maximum size for output chunks
//
// Returns array of HTML chunks that maintain structural integrity.
//
// The function works recursively:
// 1. Tries each separator in order until finding one that matches
// 2. Splits text using the matching separator
// 3. For chunks larger than chunkSize:
//   - If more separators available: recursively splits with remaining separators
//   - If no more separators: keeps as-is
//
// 4. Merges smaller chunks while respecting chunkSize
func SplitHtml(text string, separators []string, chunkSize int) []string {
	finalChunks := []string{}
	separator := separators[len(separators)-1]
	newSeparators := []string{}
	var _separator string

	// Find first matching separator
	for i, _s := range separators {
		_separator = _s
		if _s == "" {
			separator = _s
			break
		}
		if ok, _ := regexp.MatchString(_separator, text); ok {
			separator = _s
			newSeparators = separators[i+1:]
			break
		}
	}
	_separator = separator

	splits := splitTextWithRegex(text, _separator, true)
	goodSplits := []string{}
	_separator = ""

	// Process each split
	for _, s := range splits {
		if len(s) < chunkSize {
			goodSplits = append(goodSplits, s)
		} else {
			// Handle accumulated good splits before processing large split
			if len(goodSplits) > 0 {
				mergedText := mergeSplits(goodSplits, _separator, chunkSize)
				finalChunks = append(finalChunks, mergedText...)
				goodSplits = []string{}
			}

			// Process large split
			if len(newSeparators) == 0 {
				finalChunks = append(finalChunks, s)
			} else {
				other_info := SplitHtml(s, newSeparators, chunkSize)
				finalChunks = append(finalChunks, other_info...)
			}
		}
	}

	// Handle any remaining good splits
	if len(goodSplits) > 0 {
		mergedText := mergeSplits(goodSplits, _separator, chunkSize)
		finalChunks = append(finalChunks, mergedText...)
	}
	return finalChunks
}
