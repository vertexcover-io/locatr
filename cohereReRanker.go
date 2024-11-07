package locatr

import (
	"context"

	cohere "github.com/cohere-ai/cohere-go/v2"
	cohereclient "github.com/cohere-ai/cohere-go/v2/client"
)

type cohereClient struct {
	Token  string
	model  string
	client *cohereclient.Client
}

type reRankResult struct {
	Index int
	Score float64
}

type reRankRequest struct {
	Query     string
	Documents []string
}

func (c *cohereClient) reRank(request reRankRequest) (*[]reRankResult, error) {
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
		},
	)
	if err != nil {
		return nil, err
	}
	results := []reRankResult{}
	for _, doc := range response.Results {
		results = append(results, reRankResult{
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
