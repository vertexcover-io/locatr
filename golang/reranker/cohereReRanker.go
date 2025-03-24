package reranker

import (
	"context"
	"os"

	cohere "github.com/cohere-ai/cohere-go/v2"
	cohereclient "github.com/cohere-ai/cohere-go/v2/client"
	"github.com/vertexcover-io/locatr/golang/tracing"
)

var TOP_N_CHUNKS int = 8

const COHERE_RERANK_MODEL = "rerank-english-v3.0"

type ReRankInterface interface {
	ReRank(ctx context.Context, request ReRankRequest) (*[]ReRankResult, error)
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

func (c *cohereClient) ReRank(ctx context.Context, request ReRankRequest) (*[]ReRankResult, error) {
	ctx, span := tracing.StartSpan(ctx, "ReRank")
	defer span.End()

	span.AddEvent("generating request documents")
	rerankDocs := []*cohere.RerankRequestDocumentsItem{}
	for _, doc := range request.Documents {
		rerankDocs = append(rerankDocs, &cohere.RerankRequestDocumentsItem{
			String: doc,
		})
	}

	span.AddEvent("initiating rerank")
	response, err := c.client.Rerank(
		ctx,
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

	span.AddEvent("reading result")
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
