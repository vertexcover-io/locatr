// Package reranker provides document re-ranking capabilities using various backends.
package reranker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	cohere "github.com/cohere-ai/cohere-go/v2"
	cohereclient "github.com/cohere-ai/cohere-go/v2/client"
	"github.com/vertexcover-io/locatr/golang/logging"
	"github.com/vertexcover-io/locatr/golang/types"
)

// RerankerProvider constants define the supported Reranker service providers
const (
	Cohere types.RerankerProvider = "cohere"
)

type config struct {
	provider types.RerankerProvider
	model    string
	apiKey   string
	logger   *slog.Logger
}

type Option func(*config)

// WithProvider sets the reranker provider for the configuration.
func WithProvider(provider types.RerankerProvider) Option {
	return func(c *config) {
		c.provider = provider
	}
}

// WithModel sets the reranker model for the configuration.
func WithModel(model string) Option {
	return func(c *config) {
		c.model = model
	}
}

// WithAPIKey sets the reranker API key for the configuration.
func WithAPIKey(apiKey string) Option {
	return func(c *config) {
		c.apiKey = apiKey
	}
}

// WithLogger sets the logger for the reranker client.
func WithLogger(logger *slog.Logger) Option {
	return func(c *config) {
		c.logger = logger
	}
}

// rerankerClient wraps the rerank API client to implement the RerankerInterface.
// It provides document reranking capabilities using the configured reranker provider.
type rerankerClient struct {
	config  *config
	handler func(request *types.RerankRequest) ([]types.RerankResult, error)
}

// NewRerankerClient creates a new instance of the reranker client.
//
// Parameters:
//   - opts: Configuration options for the reranker client
//
// Returns:
//   - *rerankerClient: Configured client instance
//   - error: Any initialization errors
func NewRerankerClient(opts ...Option) (*rerankerClient, error) {

	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.provider == "" {
		return nil, errors.New("reranker provider is required")
	}
	if cfg.model == "" {
		return nil, errors.New("reranker model is required")
	}
	if cfg.logger == nil {
		cfg.logger = logging.DefaultLogger
	}

	var handler func(request *types.RerankRequest) ([]types.RerankResult, error)
	switch cfg.provider {
	case Cohere:
		handler = func(request *types.RerankRequest) ([]types.RerankResult, error) {
			return requestCohere(
				cohereclient.NewClient(cohereclient.WithToken(cfg.apiKey)),
				cfg.model,
				request,
			)
		}
	default:
		return nil, errors.New("invalid provider for reranker")
	}
	return &rerankerClient{config: cfg, handler: handler}, nil
}

var errDefaultRerankerAPIKeyNotSet = errors.New("'LOCATR_COHERE_API_KEY' or 'COHERE_API_KEY' environment variable is not set")

// DefaultRerankerClient returns a default reranker client using Cohere's rerank-english-v3.0 model.
//
// Parameters:
//   - logger: Logger instance for logging
//
// Returns:
//   - *rerankerClient: Configured client instance
//   - error: Any initialization errors
func DefaultRerankerClient(logger *slog.Logger) (*rerankerClient, error) {
	apiKey := os.Getenv("LOCATR_COHERE_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("COHERE_API_KEY")
		if apiKey == "" {
			return nil, errDefaultRerankerAPIKeyNotSet
		}
	}
	options := []Option{
		WithProvider(Cohere),
		WithModel("rerank-english-v3.0"),
		WithAPIKey(apiKey),
	}
	if logger != nil {
		options = append(options, WithLogger(logger))
	}
	return NewRerankerClient(options...)
}

func (client *rerankerClient) Rerank(request *types.RerankRequest) ([]types.RerankResult, error) {
	topic := fmt.Sprintf(
		"[Reranker] provider: %v, model: %v", client.config.provider, client.config.model,
	)
	defer logging.CreateTopic(topic, client.config.logger)()
	return client.handler(request)
}

// requestCohere handles API requests to Cohere's reranking API.
//
// Parameters:
//   - client: Configured Cohere API client
//   - model: Model identifier
//   - request: Contains the query and documents to rerank
//
// Returns:
//   - []RerankResult: Sorted list of documents with relevance scores
//   - error: If the reranking operation fails
func requestCohere(client *cohereclient.Client, model string, request *types.RerankRequest) ([]types.RerankResult, error) {
	// Convert documents to Cohere's expected format
	rerankDocs := []*cohere.RerankRequestDocumentsItem{}
	for _, doc := range request.Documents {
		rerankDocs = append(
			rerankDocs, &cohere.RerankRequestDocumentsItem{String: doc},
		)
	}

	// Call Cohere's rerank API
	response, err := client.Rerank(
		context.Background(),
		&cohere.RerankRequest{
			Query:     request.Query,
			Model:     &model,
			Documents: rerankDocs,
			TopN:      &request.TopN,
		},
	)
	if err != nil {
		return nil, err
	}

	// Convert Cohere response to our internal format
	results := []types.RerankResult{}
	for _, doc := range response.Results {
		results = append(results, types.RerankResult{
			Index: doc.Index,
			Score: doc.RelevanceScore,
		})
	}
	return results, nil
}
