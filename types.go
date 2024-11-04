package locatr

type IdToLocatorMap map[string][]string

// PluginInterface is an interface that defines the methods that a plugin must implement.
type PluginInterface interface {
	evaluateJsScript(scriptContent string) error
	evaluateJsFunction(function string) (string, error)
}

type LlmProvider string

type LogLevel int
