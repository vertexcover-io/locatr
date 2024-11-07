package locatr

const COHERE_RERANK_THRESHOLD = 0.9

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

const LOCATR_PROMPT = `You are provided with an HTML DOM structure and a user requirement. Your task is to find the element
in the DOM that matches the user's requirement. Each element provided in the DOM has a guaranteed unique_id.
Do not perform any other actions or modifications based on the user requirement—simply identify the element
and return its unique_id in the provided JSON format. Make sure you give valid JSON and nothing else,
you don't need to explain or add json in code-blocks, just pure-valid json is required.

Output Format:
{"locator_id": "str"}

Input Format:
{
  "html_dom": "<!-- Your HTML DOM here -->",
  "user_req": "The user's requirement here"
}`

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
