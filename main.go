package main

import (
	"encoding/json"
	"errors"
	"log"
	"unsafe"

	"github.com/vertexcover-io/locatr/llm"
	"github.com/vertexcover-io/locatr/locatr"
)

type wasmPlugin struct{}

type wasmLocatrApp struct {
	baseLocatr *locatr.BaseLocatr
}

var app *wasmLocatrApp

//export wasiEvaluateJs
func wasiEvaluateJs(ptr, length int32) (int32, int32)

//export wasiGetMemory
func wasiGetMemory() int32

func evaluateFunc(jsStr string) string {
	inputPtr := unsafe.Pointer(&[]byte(jsStr)[0])
	inputLen := int32(len(jsStr))

	resultPtr, resultLen := wasiEvaluateJs(int32(uintptr(inputPtr)), inputLen)
	memoryPtr := wasiGetMemory()

	resultSlice := (*[1 << 30]byte)(unsafe.Pointer(uintptr(memoryPtr)))[resultPtr : resultPtr+resultLen]
	return string(resultSlice)
}

func (p *wasmPlugin) LoadJsScript(scriptPath string) error {
	scriptStr, err := locatr.ReadStaticFile(scriptPath)
	if err != nil {
		return err
	}
	evaluateFunc(string(scriptStr))
	return nil
}

func (p *wasmPlugin) GetMinifiedDom() (*locatr.ElementSpec, error) {
	result := evaluateFunc("minifyHTML()")
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
	result := evaluateFunc("getElementIdLocatorMap()")
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
	result := evaluateFunc(jsFunction)
	if result == "" {
		return "", errors.New("no valid locator found")
	}
	return result, nil
}

//export InitLocatr
func InitLocatr(provider, model, apiKey string) string {
	llmClient, err := llm.NewLlmClient(provider, model, apiKey)
	if err != nil {
		return err.Error()
	}

	plugin := &wasmPlugin{}
	app = &wasmLocatrApp{
		baseLocatr: locatr.NewBaseLocatr(plugin, llmClient),
	}

	return ""
}

//export GetLocatorStr
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

func main() {
	// SO that compiler won't optimize out the functions
	wasiEvaluateJs(0, 0)
	wasiGetMemory()
}
