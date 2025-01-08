// Borrowed from: https://github.com/langchain-ai/langchain/blob/master/libs/text-splitters/langchain_text_splitters/character.py
// File contains the implementation of RecursiveCharacterTextSplitter

package reranker

import (
	"log"
	"regexp"
	"strings"
)

const CHUNK_OVERLAP = 200

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

func splitTextWithRegex(
	text string, separator string, keepSeparator bool,
) []string {
	splits := []string{}
	if separator != "" {
		if keepSeparator {
			_splits := splitWithSeparator(text, separator)
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
			// prepend the first part of the _split
			splits = append([]string{_splits[0]}, splits...)
		} else {
			splits = splitWithSeparator(text, separator)
		}
	} else {
		splits = []string{text}
	}
	finalSplits := []string{}
	for _, split := range splits {
		if split != "" {
			finalSplits = append(finalSplits, split)
		}
	}
	return finalSplits
}
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
				log.Println("Create chunk size of ", currentSize, " is greater than ", maxChunkSize)
			}
			if len(currentChunks) > 0 {
				mergedDoc := joinDocs(currentChunks, separator, false)
				if mergedDoc != nil {
					mergedDocs = append(mergedDocs, *mergedDoc)
				}

				for {
					if len(currentChunks) > 0 {
						additionalSeparatorLength = separatorLength
					} else {
						additionalSeparatorLength = 0
					}
					if !((currentSize > CHUNK_OVERLAP) ||
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

func SplitHtml(text string, separators []string, chunkSize int) []string {

	finalChunks := []string{}
	separator := separators[len(separators)-1]
	newSeparators := []string{}
	var _separator string

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
	for _, s := range splits {
		if len(s) < chunkSize {
			goodSplits = append(goodSplits, s)
		} else {
			if len(goodSplits) > 0 {
				mergedText := mergeSplits(goodSplits, _separator, chunkSize)
				finalChunks = append(finalChunks, mergedText...)
				goodSplits = []string{}
			}
			if len(newSeparators) == 0 {
				finalChunks = append(finalChunks, s)
			} else {
				other_info := SplitHtml(s, newSeparators, chunkSize)
				finalChunks = append(finalChunks, other_info...)
			}
		}
	}
	if len(goodSplits) > 0 {
		mergedText := mergeSplits(goodSplits, _separator, chunkSize)
		finalChunks = append(finalChunks, mergedText...)
	}
	return finalChunks

}
