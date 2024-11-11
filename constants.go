package locatr

const COHERE_RERANK_MODEL = "rerank-english-v3.0"

const MAX_TOKENS int = 256

const (
	OpenAI    LlmProvider = "openai"
	Anthropic LlmProvider = "anthropic"
)

// Default cache path
const DEFAULT_CACHE_PATH = ".locatr.cache"

// Default file to write locatr results
const DEFAULT_LOCATR_RESULTS_PATH = "locatr_results.json"

const (
	// Silent will not log anything
	Silent LogLevel = iota + 1
	// Error will log only errors
	Error
	// Warn will log errors and warnings
	Warn
	// Info will log errors, warnings and info
	Info
	// Debug will log everything
	Debug
)

const LOCATR_PROMPT = `Given an HTML DOM structure and a user's requirement, your task is to identify the element that matches the user's requirement and return its unique_id in a JSON format. If the element is not found, provide an appropriate error message in the JSON output. 

Ensure the JSON output is valid and nothing else is included.

Output Format:
{"locator_id": "str", "error": "str"}

Input Format:
{
  "html_dom": "<!-- Your HTML DOM here -->",
  "user_req": "The user's requirement here"
}

Process the input accordingly and ensure that if the element is not found, the "error" field contains a relevant message.`

// HTML_SEPARATORS used by html chunk splitter
var HTML_SEPARATORS = []string{
	"<body",
	"<div",
	"<p",
	"<br",
	"<li",
	"<h1",
	"<h2",
	"<h3",
	"<h4",
	"<h5",
	"<h6",
	"<span",
	"<table",
	"<tr",
	"<td",
	"<th",
	"<ul",
	"<ol",
	"<header",
	"<footer",
	"<nav",
	"<head",
	"<style",
	"<script",
	"<meta",
	"<title",
	"",
}

// CHUNK_SIZE is the maximum size of a html chunk
const CHUNK_SIZE = 4000

const CHUNK_OVERLAP = 200

var TOP_N_CHUNKS int = 8

const MAX_RETRIES_WITH_RERANK = 3

const MAX_CHUNKS_EACH_RERANK_ITERATION = 4
