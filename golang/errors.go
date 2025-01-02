package locatr

import "errors"

// BaseLocatr Errors
var (
	ErrUnableToLoadJsScripts       = errors.New("unable to load JS scripts")
	ErrUnableToMinifyHtmlDom       = errors.New("unable to minify HTML DOM")
	ErrUnableToExtractIdLocatorMap = errors.New("unable to extract ID locator map")
	ErrUnableToLocateElementId     = errors.New("unable to locate element ID")
	ErrInvalidElementIdGenerated   = errors.New("invalid element ID generated")
	ErrUnableToFindValidLocator    = errors.New("unable to find valid locator")
	ErrFailedToWriteCache          = errors.New("failed to write cache")
	ErrFailedToMarshalJson         = errors.New("failed to marshal json")
)

// LLM client errors
var ErrInvalidProviderForLlm = errors.New("invalid provider for llm")

var ErrLocatrCacheMiss = errors.New("cache miss")

var ErrLocatrRetrievalAttemptsExhausted = errors.New("failed to retrieve locatr after 3 attempts")

var ErrLocatrRetrievalFailed = errors.New("failed to retieve locatr")

var ErrNoChunksToProcess = errors.New("Got no chunks to process after reranking.")

var ErrFailedToRepariJson = errors.New("failed to repair json")
