package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"unsafe"

	"github.com/vertexcover-io/locatr/llm"
	"github.com/vertexcover-io/locatr/locatr"
)

type wasmPlugin struct{}

type wasmLocatrApp struct {
	baseLocatr *locatr.BaseLocatr
}

type getLocatorStrResult struct {
	Locator string `json:"locator,omitempty"`
	Error   string `json:"error,omitempty"`
}

type llmClient struct {
	locatr.LlmClient
	provider string
	model    string
	apiKey   string
}

const MASK uint64 = (1 << 32) - 1

var app *wasmLocatrApp

//go:wasmimport env wasiEvaluateJs
func wasiEvaluateJs(jsStrPtr uint64) uint64

//go:wasmimport env wasiHttpPost
func wasiHttpPost(urlPtr uint64, headersPtr uint64, bodyPtr uint64) uint64

func httpPost(url string, headers map[string]string, body []byte) ([]byte, error) {
	urlPtr := copyBufferToMemory([]byte(url))

	headersBytes, err := json.Marshal(headers)
	if err != nil {
		return nil, err
	}
	headersPtr := copyBufferToMemory(headersBytes)

	bodyPtr := copyBufferToMemory(body)

	resultPtr := wasiHttpPost(urlPtr, headersPtr, bodyPtr)
	resultBytes := readBufferFromMemory(resultPtr)
	return resultBytes, nil
}

func evaluateFunc(jsStr string) string {
	inputPtr := copyBufferToMemory([]byte(jsStr))
	resultPtr := wasiEvaluateJs(inputPtr)

	resultBytes := readBufferFromMemory(resultPtr)
	return string(resultBytes)
}

func (p *wasmPlugin) LoadJsScript(scriptPath string) error {
	fmt.Println("Loading script: ", scriptPath)

	scriptStr, err := locatr.ReadStaticFile(scriptPath)
	if err != nil {
		return err
	}
	evaluateFunc(string(scriptStr))
	return nil
}

func (p *wasmPlugin) GetMinifiedDom() (*locatr.ElementSpec, error) {
	fmt.Println("Getting minified dom")

	result := evaluateFunc("minifyHTML()")
	if result == "" {
		return nil, errors.New("empty minified dom")
	}

	elementSpec := &locatr.ElementSpec{}
	if err := json.Unmarshal([]byte(result), elementSpec); err != nil {
		return nil, err
	}

	return elementSpec, nil
}

func (p *wasmPlugin) ExtractIdLocatorMap() (locatr.IdToLocatorMap, error) {
	fmt.Println("Extracting id locator map")

	result := evaluateFunc("mapElementsToJson()")
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
	fmt.Println("Getting valid locator")
	// jsFunction := `
	//    (locators) => {
	//        for (const locator of locators) {
	//            try {
	//                const elements = document.querySelectorAll(locator);
	//                if (elements.length === 1) {
	//                    return locator;
	//                }
	//            } catch (error) {
	//                // If there's an error with this locator, continue to the next one
	//                continue;
	//            }
	//        }
	//        return null;
	//    }
	//  `
	// locatorsStr := fmt.Sprintf("%#v", locators)
	// jsFuncStr := fmt.Sprintf("const locators = %s; %s", locatorsStr, jsFunction)
	// result := evaluateFunc(jsFunction + '(' + locatorsStr + ')')
	// if result == "" {
	// 	return "", errors.New("no valid locator found")
	// }
	// return result, nil

	resultLocator := locators[0]
	return resultLocator, nil
}

//export InitLocatr
func InitLocatr(providerPtr, modelPtr, apiKeyPtr uint64) {
	provider := string(readBufferFromMemory(providerPtr))
	model := string(readBufferFromMemory(modelPtr))
	apiKey := string(readBufferFromMemory(apiKeyPtr))

	llmClient, _ := llm.NewLlmClient(provider, model, apiKey, httpPost)

	plugin := &wasmPlugin{}
	app = &wasmLocatrApp{
		baseLocatr: locatr.NewBaseLocatr(plugin, llmClient),
	}
}

//export GetLocatorStr
func GetLocatorStr(userReqPtr uint64) uint64 {
	userReq := string(readBufferFromMemory(userReqPtr))

	result := getLocatorStrResult{}

	if app == nil {
		result.Error = "Locator app not initialized"
	} else {
		locator, err := app.baseLocatr.GetLocatorStr(userReq)
		if err != nil {
			result.Error = err.Error()
		} else {
			result.Locator = locator
		}
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		result.Error = err.Error()
	}

	return copyBufferToMemory(jsonBytes)
}

func readBufferFromMemory(bufferPosition uint64) []byte {
	ptr := bufferPosition >> 32
	length := bufferPosition & MASK
	pointer := uintptr(ptr)

	buffer := make([]byte, length)

	for i := 0; i < int(length); i++ {
		s := *(*byte)(unsafe.Pointer(pointer + uintptr(i)))
		buffer[i] = s
	}

	return buffer
}

func copyBufferToMemory(buffer []byte) uint64 {
	bufferPtr := &buffer[0]
	unsafePtr := uintptr(unsafe.Pointer(bufferPtr))

	ptr := uint32(unsafePtr)
	size := uint32(len(buffer))

	return (uint64(ptr) << uint64(32)) | uint64(size)
}

func main() {
	wasiEvaluateJs(0)
}
