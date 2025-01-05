package reranker

import (
	"context"
	"os"

	cohere "github.com/cohere-ai/cohere-go/v2"
	cohereclient "github.com/cohere-ai/cohere-go/v2/client"
)

var TOP_N_CHUNKS int = 8

const COHERE_RERANK_MODEL = "rerank-english-v3.0"

type ReRankInterface interface {
	ReRank(request ReRankRequest) (*[]ReRankResult, error)
}

type cohereClient struct {
	Token  string
	model  string
	client *cohereclient.Client
}

type ReRankResult struct {
	Index int
	Score float64
}

type ReRankRequest struct {
	Query     string
	Documents []string
}

func (c *cohereClient) ReRank(request ReRankRequest) (*[]ReRankResult, error) {
	rerankDocs := []*cohere.RerankRequestDocumentsItem{}
	for _, doc := range request.Documents {
		rerankDocs = append(rerankDocs, &cohere.RerankRequestDocumentsItem{
			String: doc,
		})
	}
	response, err := c.client.Rerank(
		context.Background(),
		&cohere.RerankRequest{
			Query:     request.Query,
			Model:     &c.model,
			Documents: rerankDocs,
			TopN:      &TOP_N_CHUNKS,
		},
	)
	if err != nil {
		return nil, err
	}
	results := []ReRankResult{}
	for _, doc := range response.Results {
		results = append(results, ReRankResult{
			Index: doc.Index,
			Score: doc.RelevanceScore,
		})
	}
	return &results, nil
}

// NewCohereClient creates a new cohere client.
func NewCohereClient(token string) ReRankInterface {
	client := cohereclient.NewClient(cohereclient.WithToken(token))
	return &cohereClient{
		Token:  token,
		model:  COHERE_RERANK_MODEL,
		client: client,
	}
}

func CreateCohereClientFromEnv() ReRankInterface {
	apiKey := os.Getenv("COHERE_API_KEY")
	if apiKey == "" {
		return nil
	}
	return NewCohereClient(os.Getenv("COHERE_API_KEY"))
}
