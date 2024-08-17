package plugins

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"unsafe"

	"github.com/vertexcover-io/locatr/llm"
	"github.com/vertexcover-io/locatr/locatr"
)

type wasmPlugin struct {
	evaluateFunc func(string) string
}

type wasmLocatrApp struct {
	baseLocatr *locatr.BaseLocatr
}

var app *wasmLocatrApp

func (p *wasmPlugin) LoadJsScript(scriptPath string) error {
	scriptStr, err := locatr.ReadStaticFile(scriptPath)
	if err != nil {
		return err
	}
	p.evaluateFunc(string(scriptStr))
	return nil
}

func (p *wasmPlugin) GetMinifiedDom() (*locatr.ElementSpec, error) {
	result := p.evaluateFunc("minifyHTML()")
	if result == "" {
		return nil, locatr.ErrUnableToMinifyHtmlDom
	}

	elementSpec := &locatr.ElementSpec{}
	if err := json.Unmarshal([]byte(result), elementSpec); err != nil {
		return nil, locatr.ErrUnableToMinifyHtmlDom
	}

	return elementSpec, nil
}

func (p *wasmPlugin) ExtractIdLocatorMap() (locatr.IdToLocatorMap, error) {
	result := p.evaluateFunc("getElementIdLocatorMap()")
	if result == "" {
		return nil, locatr.ErrUnableToExtractIdLocatorMap
	}

	idLocatorMap := &locatr.IdToLocatorMap{}
	if err := json.Unmarshal([]byte(result), idLocatorMap); err != nil {
		return nil, err
	}
	return *idLocatorMap, nil
}

func (p *wasmPlugin) GetValidLocator(locators []string) (string, error) {
	jsFunction := `
    (locators) => {
        for (const locator of locators) {
            try {
                const elements = document.querySelectorAll(locator);
                if (elements.length === 1) {
                    return locator;
                }
            } catch (error) {
                // If there's an error with this locator, continue to the next one
                continue;
            }
        }
        return null;
    }
    `
	result := p.evaluateFunc(jsFunction)
	if result == "" {
		return "", errors.New("no valid locator found")
	}
	return result, nil
}

func InitLocatr(provider, model, apiKey string, evaluateFunc func(string) string) string {
	llmClient, err := llm.NewLlmClient(provider, model, apiKey)
	if err != nil {
		return err.Error()
	}

	plugin := &wasmPlugin{evaluateFunc: evaluateFunc}
	app = &wasmLocatrApp{
		baseLocatr: locatr.NewBaseLocatr(plugin, llmClient),
	}

	return ""
}

func GetLocatorStr(userReq string) any {
	if app == nil {
		return []interface{}{nil, errors.New("Locator app not initialized")}
	}

	locator, err := app.baseLocatr.GetLocatorStr(userReq)
	if err != nil {
		log.Println("Error getting locator string:", err)
		return []interface{}{nil, err.Error()}
	}
	return []interface{}{locator, nil}
}

func allocate(size int) uintptr {
	buf := make([]byte, size)
	return uintptr(unsafe.Pointer(&buf[0]))
}

func main() {
	fmt.Println("Wasm module loaded")
}
