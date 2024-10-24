package core

type PluginInterface interface {
	EvaluateJsScript(scriptContent string) error
	EvaluateJsFunction(function string) (string, error)
}
