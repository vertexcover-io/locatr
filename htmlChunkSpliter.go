// Borrowed from: https://github.com/langchain-ai/langchain/blob/master/libs/text-splitters/langchain_text_splitters/character.py
// File contains the implementation of RecursiveCharacterTextSplitter

package locatr

import (
	_ "fmt"
	"regexp"
	"strings"
)

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
	text string, seperator string, keepSeperator bool,
) []string {
	splits := []string{}
	if seperator != "" {
		if keepSeperator {
			_splits := splitWithSeparator(text, seperator)
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
			splits = splitWithSeparator(text, seperator)
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

func mergeSplits(parts []string, separator string, maxChunkSize int) []string {
	separatorLength := len(separator)
	resultingChunks := []string{}
	currentChunk := []string{}
	currentSize := 0

	for _, part := range parts {
		partLength := len(part)
		var additionalLength int
		if len(currentChunk) > 0 {
			additionalLength = separatorLength
		} else {
			additionalLength = 0
		}
		if currentSize+partLength+(separatorLength+additionalLength) > maxChunkSize {
			if len(currentChunk) > 0 {
				chunkString := joinDocs(currentChunk, separator, false)
				if chunkString != nil {
					resultingChunks = append(resultingChunks, *chunkString)
				}

				for {
					if len(currentChunk) > 0 {
						additionalLength = separatorLength
					} else {
						additionalLength = 0
					}
					if !((currentSize > ChunkOverlap) ||
						(currentSize+partLength+additionalLength > maxChunkSize) && (currentSize > 0)) {
						break
					}
					removedLength := 0
					if len(currentChunk) > 0 {
						removedLength = separatorLength
					}
					currentSize -= len(currentChunk[0]) + removedLength
					currentChunk = currentChunk[1:]
				}
			}

		}

		currentChunk = append(currentChunk, part)
		currentSize += partLength
		if len(currentChunk) > 0 {
			currentSize += separatorLength
		}

	}
	chunkString := joinDocs(currentChunk, separator, false)
	if chunkString != nil {
		resultingChunks = append(resultingChunks, *chunkString)
	}
	return resultingChunks
}

func splitHtml(text string, chunkSize int) []string {
	finalChunks := []string{}
	seperators := Seperators
	sepeartor := seperators[len(seperators)-1]
	newSeperators := []string{}
	var _seperator string

	for i, _s := range seperators {
		_seperator = _s
		if _s == "" {
			sepeartor = _s
			break
		}
		if ok, _ := regexp.MatchString(_seperator, text); ok {
			sepeartor = _s
			newSeperators = seperators[i+1:]
			break
		}
	}
	_seperator = sepeartor

	splits := splitTextWithRegex(text, _seperator, true)
	goodSplits := []string{}
	_seperator = ""
	for _, s := range splits {
		if len(s) < chunkSize {
			goodSplits = append(goodSplits, s)
		} else {
			if len(goodSplits) > 0 {
				mergedText := mergeSplits(goodSplits, _seperator, chunkSize)
				finalChunks = append(finalChunks, mergedText...)
				goodSplits = []string{}
			}
			if len(newSeperators) == 0 {
				finalChunks = append(finalChunks, s)
			} else {
				other_info := splitHtml(s, chunkSize)
				finalChunks = append(finalChunks, other_info...)
			}
		}
	}
	if len(goodSplits) > 0 {
		mergedText := mergeSplits(goodSplits, _seperator, chunkSize)
		finalChunks = append(finalChunks, mergedText...)
	}
	return finalChunks

}
