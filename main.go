package main

import (
	"log"
	"syscall/js"

	"github.com/vertexcover-io/locatr/llm"
	"github.com/vertexcover-io/locatr/locatr"
)

type wasmPlugin struct {
	plugin js.Value
}

type wasmLocatrApp struct {
	baseLocatr *locatr.BaseLocatr
}

var _ locatr.PluginInterface = (*wasmPlugin)(nil)

func (p *wasmPlugin) LoadJsScript(scriptPath string) error {
	result := p.plugin.Call("loadJsScript", scriptPath)
	if result.IsNull() || result.IsUndefined() {
		return locatr.ErrUnableToLoadJsScripts
	}

	if result.Type() == js.TypeString && result.String() != "" {
		return locatr.ErrUnableToLoadJsScripts
	}

	return nil
}

func (p *wasmPlugin) GetMinifiedDom() (*locatr.ElementSpec, error) {
	result := p.plugin.Call("getMinifiedDom")
	if result.IsNull() || result.IsUndefined() {
		return nil, locatr.ErrUnableToMinifyHtmlDom
	}

	var convertToElementSpec func(js.Value) locatr.ElementSpec
	convertToElementSpec = func(jsValue js.Value) locatr.ElementSpec {
		children := []locatr.ElementSpec{}
		childrenJS := jsValue.Get("children")
		for i := 0; i < childrenJS.Length(); i++ {
			children = append(children, convertToElementSpec(childrenJS.Index(i)))
		}

		attributes := make(map[string]string)
		attributesJS := jsValue.Get("attributes")
		keys := js.Global().Get("Object").Call("keys", attributesJS)
		for i := 0; i < keys.Length(); i++ {
			key := keys.Index(i).String()
			attributes[key] = attributesJS.Get(key).String()
		}

		return locatr.ElementSpec{
			ID:         jsValue.Get("id").String(),
			TagName:    jsValue.Get("tag_name").String(),
			Text:       jsValue.Get("text").String(),
			Attributes: attributes,
			Children:   children,
		}
	}

	elementSpec := convertToElementSpec(result)
	return &elementSpec, nil
}

func (p *wasmPlugin) ExtractIdLocatorMap() (locatr.IdToLocatorMap, error) {
	result := p.plugin.Call("extractIdLocatorMap")
	if result.IsNull() || result.IsUndefined() {
		return nil, locatr.ErrUnableToExtractIdLocatorMap
	}

	idToLocatorMap := make(locatr.IdToLocatorMap)
	keys := js.Global().Get("Object").Call("keys", result)
	for i := 0; i < keys.Length(); i++ {
		key := keys.Index(i).String()
		valueArray := result.Get(key)
		if valueArray.Type() != js.TypeObject || !valueArray.InstanceOf(js.Global().Get("Array")) {
			return nil, locatr.ErrUnableToExtractIdLocatorMap
		}

		values := make([]string, valueArray.Length())
		for j := 0; j < valueArray.Length(); j++ {
			values[j] = valueArray.Index(j).String()
		}

		idToLocatorMap[key] = values
	}

	return idToLocatorMap, nil
}

func (p *wasmPlugin) GetValidLocator(locators []string) (string, error) {
	jsLocators := js.ValueOf(locators)

	result := p.plugin.Call("getValidLocator", jsLocators)
	if result.IsNull() || result.IsUndefined() {
		return "", locatr.ErrUnableToFindValidLocator
	}

	if result.Type() != js.TypeString {
		return "", locatr.ErrUnableToFindValidLocator
	}

	return result.String(), nil
}

func initLocatr(this js.Value, p []js.Value) interface{} {
	llmCreds := p[0]
	llmClient, err := llm.NewLlmClient(
		llmCreds.Get("provider").String(),
		llmCreds.Get("model").String(),
		llmCreds.Get("apiKey").String(),
	)
	if err != nil {
		log.Fatalf("Failed to initialize LLM client: %v", err)
		return map[string]interface{}{
			"error": err.Error(),
		}
	}

	plugin := &wasmPlugin{plugin: p[1]}
	return &wasmLocatrApp{
		baseLocatr: locatr.NewBaseLocatr(plugin, llmClient),
	}
}

func (app *wasmLocatrApp) registerCallbacks() {
	js.Global().Set("getLocatorStr", js.FuncOf(app.getLocatorStr))
}

func (app *wasmLocatrApp) getLocatorStr(this js.Value, p []js.Value) interface{} {
	userReq := p[0].String()
	locator, err := app.baseLocatr.GetLocatorStr(userReq)
	if err != nil {
		log.Println("Error getting locator string:", err)
		return map[string]interface{}{
			"error": err.Error(),
		}
	}
	return map[string]interface{}{
		"locator": locator,
	}
}

// func main() {
// 	js.Global().Set("initLocatr", js.FuncOf(initLocatr))
// 	select {}
// }
