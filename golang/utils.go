package locatr

import (
	"errors"
	"fmt"
	"strings"

	"github.com/vertexcover-io/locatr/golang/types"
	"golang.org/x/net/html"
)

// sortRerankChunks reorders a list of text chunks based on their relevance scores.
// Parameters:
//   - chunks: Original array of text chunks to be sorted
//   - results: Array of ReRankResult containing relevance scores and indices
//
// Returns a new array containing only the valid chunks, ordered by their relevance scores.
// If no valid results are found, returns the original chunks array unchanged.
func sortRerankChunks(chunks []string, results []types.ReRankResult) []string {
	// Filter out results with indices out of range
	validResults := []types.ReRankResult{}
	for _, result := range results {
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

// extractFirstUniqueID finds and returns the first ID attribute from a top-level element in an HTML fragment.
// Parameters:
//   - htmlFragment: A string containing HTML markup to analyze
//
// Returns:
//   - string: The first ID attribute value found
//   - error: If no ID is found or if HTML parsing fails
//
// The function works by:
// 1. Wrapping the fragment in a root element for proper parsing
// 2. Creating a DOM tree from the HTML
// 3. Traversing the tree to find the first element with an ID attribute
func extractFirstUniqueID(htmlFragment string) (string, error) {
	wrappedHTML := "<root>" + htmlFragment + "</root>"
	doc, err := html.Parse(strings.NewReader(wrappedHTML))
	if err != nil {
		return "", fmt.Errorf("error parsing HTML: %w", err)
	}

	// Find the artificial root node
	var rootNode *html.Node
	var findRoot func(*html.Node) bool
	findRoot = func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == "root" {
			rootNode = n
			return true
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if findRoot(c) {
				return true
			}
		}

		return false
	}

	findRoot(doc)

	// Look for the first element with an ID
	var firstID string
	var findFirstID func(*html.Node) bool
	findFirstID = func(n *html.Node) bool {
		if n.Type == html.ElementNode {
			for _, attr := range n.Attr {
				if attr.Key == "id" {
					firstID = attr.Val
					return true
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if findFirstID(c) {
				return true
			}
		}

		return false
	}

	if rootNode != nil {
		findFirstID(rootNode)
	}

	if firstID == "" {
		return "", errors.New("no ID attribute found in the HTML fragment")
	}

	return firstID, nil
}
