// Package reranker provides document re-ranking capabilities using various backends.
package reranker

import (
	"context"
	"errors"
	"os"

	cohere "github.com/cohere-ai/cohere-go/v2"
	cohereclient "github.com/cohere-ai/cohere-go/v2/client"
	"github.com/vertexcover-io/locatr/golang/types"
)

// Default configuration values for the Cohere reranker
var (
	// TOP_N_CHUNKS specifies the maximum number of chunks to return after reranking
	TOP_N_CHUNKS int = 8
	// DEFAULT_COHERE_MODEL specifies which Cohere model to use for reranking
	DEFAULT_COHERE_MODEL string = "rerank-english-v3.0"
)

// cohereClient wraps the Cohere API client to implement the ReRankerInterface.
// It provides document reranking capabilities using Cohere's language models.
type cohereClient struct {
	instance *cohereclient.Client
}

// ReRank takes a list of documents and ranks them by relevance to a query.
// Parameters:
//   - request: Contains the query and documents to rerank
//
// Returns:
//   - []ReRankResult: Sorted list of documents with relevance scores
//   - error: If the reranking operation fails
//
// The function uses Cohere's reranking model to score and sort documents
// based on their semantic similarity to the query.
func (client *cohereClient) ReRank(request *types.ReRankRequest) ([]types.ReRankResult, error) {
	// Convert documents to Cohere's expected format
	rerankDocs := []*cohere.RerankRequestDocumentsItem{}
	for _, doc := range request.Documents {
		rerankDocs = append(rerankDocs, &cohere.RerankRequestDocumentsItem{
			String: doc,
		})
	}

	// Call Cohere's rerank API
	response, err := client.instance.Rerank(
		context.Background(),
		&cohere.RerankRequest{
			Query:     request.Query,
			Model:     &DEFAULT_COHERE_MODEL,
			Documents: rerankDocs,
			TopN:      &TOP_N_CHUNKS,
		},
	)
	if err != nil {
		return nil, err
	}

	// Convert Cohere response to our internal format
	results := []types.ReRankResult{}
	for _, doc := range response.Results {
		results = append(results, types.ReRankResult{
			Index: doc.Index,
			Score: doc.RelevanceScore,
		})
	}
	return results, nil
}

// NewCohereClient creates a new instance of the Cohere reranker client.
// Parameters:
//   - token: Cohere API authentication token
//
// Returns an initialized cohereClient ready for reranking operations.
func NewCohereClient(token string) *cohereClient {
	client := cohereclient.NewClient(cohereclient.WithToken(token))
	return &cohereClient{instance: client}
}

// CreateCohereClientFromEnv creates a new Cohere client using credentials from environment variables.
// Looks for COHERE_API_KEY in the environment.
// Returns:
//   - *cohereClient: Initialized client if successful
//   - error: If the API key is missing or invalid
func CreateCohereClientFromEnv() (*cohereClient, error) {
	apiKey := os.Getenv("COHERE_API_KEY")
	if apiKey == "" {
		return nil, errors.New("cohere API key is required")
	}
	return NewCohereClient(apiKey), nil
}
