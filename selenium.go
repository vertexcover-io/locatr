package locatr

import (
	"fmt"

	"github.com/tebeka/selenium"
)

const (
	BrowserCapability = "browserName"
)

type SeleniumOptions struct {
	Browser string
}

type seleniumPlugin struct {
	driver selenium.WebDriver
}

type seleniumLocatr struct {
	driver selenium.WebDriver
	locatr *BaseLocatr
}

func NewRemoteSeleniumLocatr(serverUrl string, opt BaseLocatrOptions, opts ...SeleniumOptions) (*seleniumLocatr, error) {
	caps := selenium.Capabilities{BrowserCapability: "chrome"}
	for _, opt := range opts {
		if opt.Browser != "" {
			caps[BrowserCapability] = opt.Browser
		}
	}

	wd, err := selenium.NewRemote(caps, serverUrl)
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

func (sl *seleniumLocatr) Close() error {
	return sl.driver.Quit()
}

func (pl *seleniumPlugin) evaluateJsFunction(function string) (string, error) {
	result, err := pl.driver.ExecuteScript(function, nil)
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

func (pl *seleniumPlugin) evaluateJsScript(script string) error {
	_, err := pl.driver.ExecuteScript(script, nil)
	if err != nil {
		return fmt.Errorf("failed to evaluate JS script: %w", err)
	}
	return nil
}

func documentReadyCondition(driver selenium.WebDriver) (bool, error) {
	state, err := driver.ExecuteScript("return document.readyState", nil)
	if err != nil {
		return false, err
	}
	return state == "complete", nil
}

func (pl *seleniumLocatr) GetLocatrStr(userReq string) (string, error) {
	err := pl.driver.Wait(documentReadyCondition)
	if err != nil {
		return "", fmt.Errorf("error waiting for load state: %w", err)
	}

	locatorStr, err := pl.locatr.getLocatorStr(userReq)
	if err != nil {
		return "", fmt.Errorf("error getting locator string: %w", err)
	}
	return locatorStr, nil
}
