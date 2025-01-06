package seleniumLocatr

import (
	"encoding/json"
	"errors"
	"fmt"

	locatr "github.com/vertexcover-io/locatr/golang"
	"github.com/vertexcover-io/locatr/golang/baseLocatr"
	"github.com/vertexcover-io/locatr/golang/elementSpec"
	"github.com/vertexcover-io/selenium"
)

type seleniumPlugin struct {
	driver selenium.WebDriver
}

type seleniumLocatr struct {
	driver selenium.WebDriver
	locatr *baseLocatr.BaseLocatr
}

var ErrUnableToLoadJsScriptSelenium = errors.New("unable to load js script through playwright")

// NewRemoteConnSeleniumLocatr Create a new selenium locatr with selenium seesion.
func NewRemoteConnSeleniumLocatr(serverUrl string,
	sessionId string,
	opt baseLocatr.BaseLocatrOptions,
) (*seleniumLocatr, error) {
	wd, err := selenium.ConnectRemote(serverUrl, sessionId)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to remote selenium instance: %w", err)
	}

	seleniumPlugin := seleniumPlugin{driver: wd}
	locatr := seleniumLocatr{
		driver: wd,
		locatr: baseLocatr.NewBaseLocatr(&seleniumPlugin, opt),
	}
	return &locatr, nil
}

func NewSeleniumLocatr(driver selenium.WebDriver,
	options baseLocatr.BaseLocatrOptions) (*seleniumLocatr, error) {
	plugin := &seleniumPlugin{
		driver: driver,
	}
	baseLocatr := baseLocatr.NewBaseLocatr(plugin, options)
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
	locatorStr, err := sl.locatr.GetLocatorStr(userReq)
	if err != nil {
		return "", fmt.Errorf("error getting locator string: %w", err)
	}
	return locatorStr, nil
}

func (pl *seleniumLocatr) WriteResultsToFile() {
	pl.locatr.WriteLocatrResultsToFile()
}

func (pl *seleniumLocatr) GetLocatrResults() []baseLocatr.LocatrResult {
	return pl.locatr.GetLocatrResults()
}

func (sl *seleniumPlugin) GetMinifiedDomAndLocatorMap() (
	*elementSpec.ElementSpec,
	*elementSpec.IdToLocatorMap,
	error,
) {
	if err := sl.evaluateJsScript(locatr.HTML_MINIFIER_JS_CONTENT); err != nil {
		return nil, nil, fmt.Errorf("%v : %v", ErrUnableToLoadJsScriptSelenium, err)
	}
	result, err := sl.evaluateJsFunction("minifyHTML()")
	if err != nil {
		return nil, nil, err
	}
	elementsSpec := &elementSpec.ElementSpec{}
	if err := json.Unmarshal([]byte(result), elementsSpec); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal ElementSpec json: %v", err)
	}

	result, _ = sl.evaluateJsFunction("mapElementsToJson()")
	idLocatorMap := &elementSpec.IdToLocatorMap{}
	if err := json.Unmarshal([]byte(result), idLocatorMap); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal IdToLocatorMap json: %v", err)
	}
	return elementsSpec, idLocatorMap, nil
}

func (sl *seleniumPlugin) GetCurrentContext() string {
	if value, err := sl.evaluateJsFunction("window.location.href"); err == nil {
		return value
	} else {
		return ""
	}
}

func (sl *seleniumPlugin) IsValidLocator(locatr string) (bool, error) {
	value, err := sl.evaluateJsFunction(fmt.Sprintf("isValidLocator('%s')", locatr))
	if value == "true" && err == nil {
		return true, nil
	} else {
		return false, err
	}
}
