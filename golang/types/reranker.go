package types

// ReRankRequest represents a request to re-rank a set of documents based on a query.
type ReRankRequest struct {
	Query     string   // The query string used for re-ranking
	Documents []string // List of documents to be re-ranked
}

// ReRankResult represents the result of a re-ranking operation.
type ReRankResult struct {
	Index int     // Index of the document in the original list
	Score float64 // Relevance score assigned to the document
}

// ReRankerInterface defines the interface for a re-ranker that can re-rank documents.
type ReRankerInterface interface {
	ReRank(request *ReRankRequest) ([]ReRankResult, error) // Re-rank documents based on the request
}
