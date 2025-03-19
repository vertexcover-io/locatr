package plugins

import (
	"errors"
	"fmt"
	"strings"

	"github.com/vertexcover-io/locatr/golang/internal/appium"
	"github.com/vertexcover-io/locatr/golang/internal/constants"
	"github.com/vertexcover-io/locatr/golang/internal/utils"
	"github.com/vertexcover-io/locatr/golang/internal/xml"
	"github.com/vertexcover-io/locatr/golang/types"
)

type appiumPlugin struct {
	client *appium.Client
}

func NewAppiumPlugin(serverUrl, sessionId string) (*appiumPlugin, error) {
	client, err := appium.NewClient(serverUrl, sessionId)
	if err != nil {
		return nil, err
	}
	return &appiumPlugin{client: client}, nil
}

func (plugin *appiumPlugin) evaluateJSExpression(expression string, args ...any) (any, error) {
	// Check if script is already attached
	isAttached, err := plugin.client.ExecuteScript("return window.locatrScriptAttached === true", []any{})
	if err != nil || isAttached == nil || !isAttached.(bool) {
		_, err := plugin.client.ExecuteScript(constants.JS_CONTENT, []any{})
		if err != nil {
			return nil, fmt.Errorf("could not add JS content: %v", err)
		}
	}

	result, err := plugin.client.ExecuteScript(fmt.Sprintf("return %s", expression), args)
	if err != nil {
		return nil, fmt.Errorf("error evaluating `%v` expression: %v", expression, err)
	}
	return result, nil
}

func (plugin *appiumPlugin) minifyHTML() (*types.DOM, error) {
	result, err := plugin.evaluateJSExpression("minifyHTML()")
	if err != nil {
		return nil, fmt.Errorf("couldn't get minified DOM: %v", err)
	}

	rootElement, err := utils.ParseElementSpec(result)
	if err != nil {
		return nil, err
	}

	result, err = plugin.evaluateJSExpression("createLocatorMap()")
	if err != nil {
		return nil, fmt.Errorf("couldn't get locator map: %v", err)
	}

	locatorMap, err := utils.ParseLocatorMap(result)
	if err != nil {
		return nil, err
	}

	dom := &types.DOM{
		RootElement: rootElement,
		Metadata: &types.DOMMetadata{
			LocatorType: types.CssSelectorType, LocatorMap: locatorMap,
		},
	}
	return dom, nil
}

func (plugin *appiumPlugin) minifyXML() (*types.DOM, error) {
	pageSource, err := plugin.client.GetPageSource()
	if err != nil {
		return nil, err
	}
	capabilities, err := plugin.client.GetCapabilities()
	if err != nil {
		return nil, err
	}
	platFormName := capabilities.Value.PlatformName
	if platFormName == "" {
		platFormName = capabilities.Value.Cap.PlatformName
	}
	eSpec, err := xml.MinifySource(pageSource, platFormName)
	if err != nil {
		return nil, err
	}
	locatrMap, err := xml.MapElementsToJson(pageSource, platFormName)
	if err != nil {
		return nil, err
	}
	dom := &types.DOM{
		RootElement: eSpec,
		Metadata: &types.DOMMetadata{
			LocatorType: types.XPathType, LocatorMap: locatrMap,
		},
	}
	return dom, nil
}

func (plugin *appiumPlugin) WaitForLoadEvent(timeout *float64) error {
	return nil
}

func (plugin *appiumPlugin) GetCurrentContext() (*string, error) {
	caps, err := plugin.client.GetCapabilities()
	if err != nil {
		return nil, err
	}
	platform := caps.Value.PlatformName
	if strings.ToLower(platform) != "android" {
		return nil, fmt.Errorf("cannot read platform '%s' current context", platform)
	}
	if currentActivity, err := plugin.client.GetCurrentActivity(); err != nil {
		return &currentActivity, nil
	}
	return nil, errors.New("no context found")
}

func (plugin *appiumPlugin) GetMinifiedDOM() (*types.DOM, error) {
	if plugin.client.IsWebView() {
		return plugin.minifyHTML()
	}
	return plugin.minifyXML()
}

func (plugin *appiumPlugin) IsLocatorValid(locator string) (bool, error) {
	if err := plugin.client.FindElement(locator); err == nil {
		return true, nil
	} else {
		return false, err
	}
}

func (plugin *appiumPlugin) SetViewportSize(width, height int) error {
	return errors.New("not implemented")
}

func (plugin *appiumPlugin) TakeScreenshot() ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (plugin *appiumPlugin) GetElementLocators(location *types.Location) ([]string, error) {
	return nil, errors.New("not implemented")
}

func (plugin *appiumPlugin) GetElementLocation(locator string) (*types.Location, error) {
	return nil, errors.New("not implemented")
}
