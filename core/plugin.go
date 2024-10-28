package core

type PluginInterface interface {
	evaluateJsScript(scriptContent string) error
	evaluateJsFunction(function string) (string, error)
}
