package appiumLocatr

import (
	"strings"

	locatr "github.com/vertexcover-io/locatr/golang"
	"github.com/vertexcover-io/locatr/golang/appium/appiumClient"
	"github.com/vertexcover-io/locatr/golang/elementSpec"
)

type appiumPlugin struct {
	client *appiumClient.AppiumClient
}

type appiumLocatr struct {
	locatr *locatr.BaseLocatr
}

func NewAppiumLocatr(serverUrl string, sessionId string, opts locatr.BaseLocatrOptions) (*appiumLocatr, error) {
	apC, err := appiumClient.NewAppiumClient(serverUrl, sessionId)
	if err != nil {
		return nil, err
	}
	plugin := &appiumPlugin{
		client: apC,
	}
	baseLocatr := locatr.NewBaseLocatr(plugin, opts)
	locatr := &appiumLocatr{
		locatr: baseLocatr,
	}
	return locatr, nil
}

func (apPlugin *appiumPlugin) GetMinifiedDomAndLocatorMap() (
	*elementSpec.ElementSpec,
	*elementSpec.IdToLocatorMap,
	locatr.SelectorType,
	error,
) {
	pageSource, err := apPlugin.client.GetPageSource()
	if err != nil {
		return nil, nil, "", err
	}
	capabilities, err := apPlugin.client.GetCapabilities()
	if err != nil {
		return nil, nil, "", err
	}
	platFormName := capabilities.Value.PlatformName
	if platFormName == "" {
		platFormName = capabilities.Value.Cap.PlatformName
	}
	eSpec, err := minifySource(pageSource, platFormName)
	if err != nil {
		return nil, nil, "", err
	}
	locatrMap, err := mapElementsToJson(pageSource, platFormName)
	if err != nil {
		return nil, nil, "", err
	}
	return eSpec, locatrMap, "xpath", nil
}

func (apPlugin *appiumPlugin) GetCurrentContext() string {
	capabilities, err := apPlugin.client.GetCapabilities()
	if err != nil {
		return ""
	}
	if strings.ToLower(capabilities.Value.PlatformName) != "andriod" {
		return ""
	}
	if currentActivity, err := apPlugin.client.GetCurrentActivity(); err != nil {
		return currentActivity
	}
	return ""
}

func (apPlugin *appiumPlugin) IsValidLocator(locatr string) (bool, error) {
	if err := apPlugin.client.FindElement(locatr); err == nil {
		return true, nil
	} else {
		return false, err
	}
}

func (apLocatr *appiumLocatr) GetLocatrStr(userReq string) (*locatr.LocatrOutput, error) {
	locatrOutput, err := apLocatr.locatr.GetLocatorStr(userReq)
	if err != nil {
		return nil, err
	}
	return locatrOutput, nil

}
func (apLocatr *appiumLocatr) WriteResultsToFile() {
	apLocatr.locatr.WriteLocatrResultsToFile()
}

func (apLocatr *appiumLocatr) GetLocatrResults() []locatr.LocatrResult {
	return apLocatr.locatr.GetLocatrResults()
}
