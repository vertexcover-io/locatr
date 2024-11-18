package main

import (
	"C"
	"encoding/json"
	"fmt"

	"github.com/vertexcover-io/locatr"
	_ "unsafe"
)

var BaseLocatrs = make([]*locatr.BaseLocatr, 0)

//export CreateRpccConnection
func CreateRpccConnection(port int, targetId *C.char) (int, *C.char) {
	validString := C.GoString(targetId)
	index, err := locatr.CreateRpccConnection(port, validString)
	if err != nil {
		return -1, C.CString(err.Error())
	}
	return index, nil

}

//export CloseRpccConnection
func CloseRpccConnection(index int) *C.char {
	err := locatr.CloseRpccConnection(index)
	if err != nil {
		return C.CString(err.Error())
	}
	return nil
}

//export CreateBaseLocatr
func CreateBaseLocatr(
	connectionId int,
	cachePath *C.char,
	useCache bool,
	logLevel int,
	resultsFilePath *C.char,
	llmApiKey *C.char,
	llmProvider *C.char,
	llmModel *C.char,
	cohereReRankerApiKey *C.char,
) (int, *C.char) {
	var provider locatr.LlmProvider = locatr.LlmProvider(C.GoString(llmProvider))
	llmClinet, err := locatr.NewLlmClient(
		provider,
		C.GoString(llmModel),
		C.GoString(llmApiKey),
	)
	if err != nil {
		return -1, C.CString(err.Error())
	}
	reRankerClient := locatr.NewCohereClient(
		C.GoString(cohereReRankerApiKey),
	)
	options := locatr.BaseLocatrOptions{
		CachePath: C.GoString(cachePath),
		UseCache:  useCache,
		LogConfig: locatr.LogConfig{
			Level: locatr.LogLevel(logLevel),
		},
		ResultsFilePath: C.GoString(resultsFilePath),
		ConnectionId:    connectionId,
		LlmClient:       llmClinet,
		ReRankClient:    reRankerClient,
	}
	baseLocatr, err := locatr.NewBaseLocatr(options)
	if err != nil {
		return -1, C.CString(err.Error())
	}
	BaseLocatrs = append(BaseLocatrs, baseLocatr)
	return len(BaseLocatrs) - 1, C.CString("")
}

//export GetLocatrString
func GetLocatrString(index int, userRequest *C.char) *C.char {
	if index < 0 || index >= len(BaseLocatrs) {
		return C.CString(fmt.Sprintf("Invalid index to get base locatr %d", index))
	}
	baselocatr := BaseLocatrs[index]
	if baselocatr == nil {
		return C.CString(fmt.Sprintf("Base locatr at index %d is nil", index))
	}
	locatrString, err := baselocatr.GetLocatorStr(C.GoString(userRequest))
	if err != nil {
		return C.CString(err.Error())
	}
	return C.CString(locatrString)
}

//export WriteLocatrResults
func WriteLocatrResults(
	index int,
) *C.char {
	if index < 0 || index >= len(BaseLocatrs) {
		return C.CString(fmt.Sprintf("Invalid index to get base locatr %d", index))
	}
	baselocatr := BaseLocatrs[index]
	if baselocatr == nil {
		return C.CString(fmt.Sprintf("Base locatr at index %d is nil", index))
	}
	baselocatr.WriteLocatrResultsToFile()
	return nil
}

//export GetLocatrResults
func GetLocatrResults(
	index int,
) (*C.char, *C.char) { // first is the json string & second is the
	if index < 0 || index >= len(BaseLocatrs) {
		return nil, C.CString(fmt.Sprintf("Invalid index to get base locatr %d", index))
	}
	baselocatr := BaseLocatrs[index]
	if baselocatr == nil {
		return nil, C.CString(fmt.Sprintf("Base locatr at index %d is nil", index))
	}
	results := baselocatr.GetLocatrResults()
	resultsJson, err := json.Marshal(results)
	if err != nil {
		return nil, C.CString(err.Error())
	}
	return C.CString(string(resultsJson)), nil
}

func main() {}
