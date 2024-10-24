package locatr

type PluginInterface interface {
	EvaluateJsScript(scriptContent string) error
	EvaluateJsFunction(function string) (string, error)
}
