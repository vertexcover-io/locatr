package types

// RerankerProvider represents the provider of the reranker.
type RerankerProvider string

// RerankRequest represents a request to re-rank a set of documents based on a query.
type RerankRequest struct {
	Query     string   // The query string used for re-ranking
	Documents []string // List of documents to be re-ranked
	TopN      int      // Maximum number of results to return
}

// RerankResult represents the result of a re-ranking operation.
type RerankResult struct {
	Index int     // Index of the document in the original list
	Score float64 // Relevance score assigned to the document
}

// RerankerClientInterface defines the interface for a reranker client that can re-rank documents.
type RerankerClientInterface interface {
	Rerank(request *RerankRequest) ([]RerankResult, error) // Re-rank documents based on the request
}
