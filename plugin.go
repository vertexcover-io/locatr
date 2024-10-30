package locatr

// PluginInterface is an interface that defines the methods that a plugin must implement.
type PluginInterface interface {
	evaluateJsScript(scriptContent string) error
	evaluateJsFunction(function string) (string, error)
}
