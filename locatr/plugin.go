package locatr

type PluginInterface interface {
	LoadJsScript(scriptPath string) error
	GetMinifiedDom() (*ElementSpec, error)
	ExtractIdLocatorMap() (IdToLocatorMap, error)
	GetValidLocator(locators []string) (string, error)
}
