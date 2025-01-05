package appiumLocatr

import "github.com/vertexcover-io/locatr/golang/baseLocatr"

type appiumPlugin struct {
	client *appiumClinet
}

type appiumLocatr struct {
	locatr *baseLocatr.BaseLocatr
}

func NewAppiumLocatr(serverUrl string, sessionId string, opts baseLocatr.BaseLocatrOptions) (*appiumLocatr, error) {
	appiumClinet, err := newAppiumClient(serverUrl, sessionId)
	if err != nil {
		return nil, err
	}
	plugin := &appiumPlugin{
		client: appiumClinet,
	}
	baseLocatr := baseLocatr.NewBaseLocatr(plugin, opts)
	return &appiumLocatr{
		locatr: baseLocatr,
	}, nil
}

func (ac *appiumPlugin) getPageSource() string {
	return ac.getPageSource()
}
