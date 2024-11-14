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

const LOCATR_PROMPT = `Your task is to identify the HTML element that matches a user's requirement from a given HTML DOM structure and return its unique_id in a JSON format. If the element is not found, provide an appropriate error message in the JSON output.

Each HTML element may contain an attribute called "data-supported-primitives" which indicates its supported interactions. The following attributes determine whether an element is "clickable", "hoverable", "inputable", or "selectable":

1. "clickable": The element supports click interactions and will have "data-supported-primitives" set to "click".
2. "hoverable": The element supports hover interactions and will have "data-supported-primitives" set to "hover".
3. "inputable": The element supports text input interactions and will have "data-supported-primitives" set to "input_text". If this attribute is not present then the input is read-only.
4. "selectable": The element supports selecting options and will have "data-supported-primitives" set to "select_option".

Your output should be in valid JSON format and contain only the required fields.

Output Format:
{
  "locator_id": "str",     // The unique_id of the element that matches the user's requirement.
  "error": "str"           // An appropriate error message if the element is not found.
}

Input Format:
{
  "html_dom": "<!-- Your HTML DOM here -->",
  "user_req": "The user's requirement here"
}
Process the input accordingly and ensure that if the element is not found, the “error” field contains a relevant message.
`

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
