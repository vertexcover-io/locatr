package ipc

import (
	"fmt"

	"github.com/vertexcover-io/locatr/locatr"
)

type ipcLocatr struct {
	locatr *locatr.BaseLocatr
	plugin *ipcPlugin
}

type ipcLocatorConfig struct {
	locatr.LocatrConfig
	ConnTarget string `json:"conn_target"`
}

type ipcPlugin struct {
	locatr.PluginInterface
	requestChan  chan any
	responseChan chan *ipcResponse
}

func NewIpcLocator(conf *locatr.LocatrConfig, requestChan chan any, responseChan chan *ipcResponse) (*ipcLocatr, error) {
	tcpPlugin := &ipcPlugin{
		requestChan:  requestChan,
		responseChan: responseChan,
	}

	baseLocatr, err := locatr.NewBaseLocatr(tcpPlugin, conf)
	if err != nil {
		return nil, err
	}

	return &ipcLocatr{
		locatr: baseLocatr,
		plugin: tcpPlugin,
	}, nil
}

func (tp *ipcPlugin) EvaluateJs(jsStr string) string {
	req := &ipcRequest{
		ConnType: serverRequestType,
		Method:   methodEvaluateJs,
		Params:   map[string]any{"js_str": jsStr},
	}

	tp.requestChan <- req

	resp := <-tp.responseChan

	if resp.Error != "" || resp.StatusCode != statusOk {
		return fmt.Sprintf("Server error: %s", resp.Error)
	}

	return resp.Data.(string)
}
func (rl *ipcLocatr) GetLocator(userReq string) (string, error) {
	return rl.locatr.GetLocatorStr(userReq)
}

func (rl *ipcLocatr) Close() error {
	close(rl.plugin.requestChan)
	close(rl.plugin.responseChan)
	return nil
}
