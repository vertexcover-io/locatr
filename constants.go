package locatr

var CohereReRankThreshold = 0.8

var CohereReRankModel = "rerank-english-v3.0"

const MaxTokens int = 256

const (
	OpenAI    LlmProvider = "openai"
	Anthropic LlmProvider = "anthropic"
)

// Default cache path
var DEFAULT_CACHE_PATH = ".locatr.cache"

// Default file to write locatr results
var DEFAULT_LOCATR_RESULTS_PATH = "locatr_results.json"

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

const locatrPrompt = `You are provided with an HTML DOM structure and a user requirement. Your task is to find the element
in the DOM that matches the user's requirement. Each element provided in the DOM has a guranteed unique_id.
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
