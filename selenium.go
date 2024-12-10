package locatr

import (
	"fmt"

	"github.com/vertexcover-io/selenium"
)

type seleniumPlugin struct {
	driver selenium.WebDriver
}

type seleniumLocatr struct {
	driver selenium.WebDriver
	locatr *BaseLocatr
}

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

func (sl *seleniumLocatr) Close() error {
	return sl.driver.Quit()
}

func (pl *seleniumPlugin) evaluateJsFunction(function string) (string, error) {
	r_function := "return " + function
	result, err := pl.driver.ExecuteScript(r_function, nil)
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

func (pl *seleniumLocatr) GetLocatrStr(userReq string) (string, error) {
	// err := pl.driver.Wait(elementLoaded(userReq.locatr, locator_type string))
	// if err != nil {
	// 	return "", fmt.Errorf("error waiting for load state: %w", err)
	// }
	//
	locatorStr, err := pl.locatr.getLocatorStr(userReq)
	if err != nil {
		return "", fmt.Errorf("error getting locator string: %w", err)
	}
	return locatorStr, nil
}
