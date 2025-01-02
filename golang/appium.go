package locatr

type appiumPlugin struct {
	client *appiumClinet
}

type appiumLocatr struct {
	locatr *BaseLocatr
}

func NewAppiumLocatr(serverUrl string, sessionId string, opts BaseLocatrOptions) (*appiumLocatr, error) {
	appiumClinet, err := newAppiumClient(serverUrl, sessionId)
	if err != nil {
		return nil, err
	}
	plugin := &appiumPlugin{
		client: appiumClinet,
	}
	baseLocatr := NewBaseLocatr(plugin, opts)
	return &appiumLocatr{
		locatr: baseLocatr,
	}, nil
}

func (ac *appiumPlugin) getPageSource() string {
	return ac.getPageSource()
}
