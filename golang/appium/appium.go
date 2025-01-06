package appiumLocatr

import (
	"github.com/vertexcover-io/locatr/golang/appium/appiumClient"
	"github.com/vertexcover-io/locatr/golang/baseLocatr"
	"github.com/vertexcover-io/locatr/golang/elementSpec"
	"strings"
)

type appiumPlugin struct {
	client *appiumClient.AppiumClient
}

type appiumLocatr struct {
	locatr *baseLocatr.BaseLocatr
}

func NewAppiumLocatr(serverUrl string, sessionId string, opts baseLocatr.BaseLocatrOptions) (*appiumLocatr, error) {
	apC, err := appiumClient.NewAppiumClient(serverUrl, sessionId)
	if err != nil {
		return nil, err
	}
	plugin := &appiumPlugin{
		client: apC,
	}
	baseLocatr := baseLocatr.NewBaseLocatr(plugin, opts)
	locatr := &appiumLocatr{
		locatr: baseLocatr,
	}
	return locatr, nil
}

func (apPlugin *appiumPlugin) GetMinifiedDomAndLocatorMap() (
	*elementSpec.ElementSpec,
	*elementSpec.IdToLocatorMap,
	error,
) {
	pageSource, err := apPlugin.client.GetPageSource()
	if err != nil {
		return nil, nil, err
	}
	capabilities, err := apPlugin.client.GetCapabilities()
	if err != nil {
		return nil, nil, err
	}
	platform := strings.ToLower(capabilities.Value.PlatformName)
	eSpec, err := minifySource(pageSource, platform)
	if err != nil {
		return nil, nil, err
	}
	locatrMap, err := mapElementsToJson(pageSource, platform)
	if err != nil {
		return nil, nil, err
	}
	return eSpec, locatrMap, nil
}

func (apPlugin *appiumPlugin) GetCurrentContext() string {
	if currentActivity, err := apPlugin.client.GetCurrentActivity(); err != nil {
		return currentActivity
	}
	return ""
}

func (apPlugin *appiumPlugin) IsValidLocator(locatr string) (string, error) {
	if err := apPlugin.client.FindElement(locatr); err != nil {
		return "true", nil
	} else {
		return "", err
	}
}

func (apLocatr *appiumLocatr) GetLocatrStr(userReq string) (string, error) {
	locatrStr, err := apLocatr.locatr.GetLocatorStr(userReq)
	if err != nil {
		return "", err
	}
	return locatrStr, nil

}
func (apLocatr *appiumLocatr) WriteResultsToFile() {
	apLocatr.locatr.WriteLocatrResultsToFile()
}

func (apLocatr *appiumLocatr) GetLocatrResults() []baseLocatr.LocatrResult {
	return apLocatr.locatr.GetLocatrResults()
}
