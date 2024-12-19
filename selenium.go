package locatr

import (
	"fmt"
	"github.com/vertexcover-io/selenium"
)

type seleniumPlugin struct {
	driver selenium.WebDriver
	PluginInterface
}

type seleniumLocatr struct {
	driver selenium.WebDriver
	locatr *BaseLocatr
}

// NewRemoteConnSeleniumLocatr Create a new selenium locatr with selenium seesion.
func NewRemoteConnSeleniumLocatr(serverUrl string, sessionId string, opt BaseLocatrOptions) (*seleniumLocatr, error) {
	wd, err := selenium.ConnectRemote(serverUrl, sessionId)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to remote selenium instance: %w", err)
	}

	seleniumPlugin := seleniumPlugin{driver: wd}
	locatr := seleniumLocatr{
		driver: wd,
		locatr: NewBaseLocatr(&seleniumPlugin, opt),
	}

	return &locatr, nil
}

func NewSeleniumLocatr(driver selenium.WebDriver, options BaseLocatrOptions) (*seleniumLocatr, error) {
	plugin := &seleniumPlugin{
		driver: driver,
	}
	baseLocatr := NewBaseLocatr(plugin, options)
	return &seleniumLocatr{
		driver: driver,
		locatr: baseLocatr,
	}, nil

}

// Close close the selenium session.
func (sl *seleniumLocatr) Close() error {
	return sl.driver.Quit()
}

func (sl *seleniumPlugin) evaluateJsFunction(function string) (string, error) {
	rFunction := "return " + function
	result, err := sl.driver.ExecuteScript(rFunction, nil)
	if err != nil {
		return "", fmt.Errorf("failed to evaluate JS function: %w", err)
	}
	if result == nil {
		return "", fmt.Errorf("failed to obtain result from function")
	}

	switch res := result.(type) {
	case string:
		return res, nil
	default:
		return fmt.Sprint(res), nil
	}
}

func (sl *seleniumPlugin) evaluateJsScript(script string) error {
	_, err := sl.driver.ExecuteScript(script, nil)
	if err != nil {
		return fmt.Errorf("failed to evaluate JS script: %w", err)
	}
	return nil
}

func (sl *seleniumLocatr) GetLocatrStr(userReq string) (string, error) {
	locatorStr, err := sl.locatr.getLocatorStr(userReq)
	if err != nil {
		return "", fmt.Errorf("error getting locator string: %w", err)
	}
	return locatorStr, nil
}
