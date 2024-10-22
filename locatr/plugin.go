package locatr

type PluginInterface interface {
	EvaluateJsScript(scriptContent string) error
	GetMinifiedDom() (*ElementSpec, error)
	ExtractIdLocatorMap() (IdToLocatorMap, error)
	GetValidLocator(locators []string) (string, error)
}
