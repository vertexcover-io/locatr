package types

import "log/slog"

// CacheEntry represents a cache entry for storing locator information.
type CacheEntry struct {
	UserRequest string      `json:"user_request"` // User's request or query name
	Locators    []string    `json:"locators"`     // List of locators associated with the request
	LocatorType locatorType `json:"locator_type"` // Type of locator used
}

// LocatrCompletion represents the completion result of Locate method.
type LocatrCompletion struct {
	Locators    []string    `json:"locators"`     // List of locators found, all of them point to the same element
	LocatorType locatorType `json:"locator_type"` // Type of locators in the list
	CacheHit    bool        `json:"cache_hit"`    // Indicates if the result was a cache hit
	LLMCompletionMeta
}

// LocatrMode defines the interface for a Locatr mode to use for processing requests.
type LocatrMode interface {
	ProcessRequest(
		request string,
		plugin PluginInterface,
		llmClient LLMClientInterface,
		rerankerClient RerankerClientInterface,
		logger *slog.Logger,
		completion *LocatrCompletion,
	) error
}
