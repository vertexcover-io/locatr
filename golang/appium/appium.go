package appiumLocatr

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	locatr "github.com/vertexcover-io/locatr/golang"
	"github.com/vertexcover-io/locatr/golang/appium/appiumClient"
	"github.com/vertexcover-io/locatr/golang/elementSpec"
	"github.com/vertexcover-io/locatr/golang/minifier"
	"github.com/vertexcover-io/locatr/golang/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type appiumPlugin struct {
	client *appiumClient.AppiumClient
}

type appiumLocatr struct {
	locatr *locatr.BaseLocatr
}

func NewAppiumLocatr(
	ctx context.Context,
	serverUrl string,
	sessionId string,
	opts locatr.BaseLocatrOptions,
) (*appiumLocatr, error) {
	span := trace.SpanFromContext(ctx)

	span.AddEvent("Connecting to remote appium instance")

	apC, err := appiumClient.NewAppiumClient(ctx, serverUrl, sessionId)
	if err != nil {
		return nil, err
	}

	span.AddEvent("Connected to remote appium instance")

	plugin := &appiumPlugin{
		client: apC,
	}
	baseLocatr := locatr.NewBaseLocatr(plugin, opts)
	locatr := &appiumLocatr{
		locatr: baseLocatr,
	}
	return locatr, nil
}

func (apPlugin *appiumPlugin) GetMinifiedDomAndLocatorMap(ctx context.Context) (
	*elementSpec.ElementSpec,
	*elementSpec.IdToLocatorMap,
	locatr.SelectorType,
	error,
) {
	ctx, span := tracing.StartSpan(ctx, "GetMinifiedDomAndLocatorMap")
	defer span.End()

	if apPlugin.client.IsWebView(ctx) {
		return apPlugin.htmlMinification(ctx)
	}
	return apPlugin.xmlMinification(ctx)
}

func (apPlugin *appiumPlugin) evaluateJsScript(ctx context.Context, script string) error {
	_, err := apPlugin.client.ExecuteScript(ctx, script, nil)
	if err != nil {
		return fmt.Errorf("failed to evaluate JS script: %w", err)
	}
	return nil
}

func (apPlugin *appiumPlugin) evaluateJsFunction(ctx context.Context, function string) (string, error) {
	rFunction := "return " + function
	result, err := apPlugin.client.ExecuteScript(ctx, rFunction, nil)
	if err != nil {
		return "", fmt.Errorf("failed to evaluate JS function: %w", err)
	}
	if result == nil {
		return "", fmt.Errorf("failed to obtain result from function")
	}

	switch res := result.(type) {
	case string:
		return res, nil
	case []byte:
		return string(res), nil
	default:
		return fmt.Sprint(res), nil
	}
}

func (apPlugin *appiumPlugin) htmlMinification(ctx context.Context) (*elementSpec.ElementSpec, *elementSpec.IdToLocatorMap, locatr.SelectorType, error) {
	if err := apPlugin.evaluateJsScript(ctx, locatr.HTML_MINIFIER_JS_CONTENT); err != nil {
		return nil, nil, "", fmt.Errorf("%v", err)
	}
	result, err := apPlugin.evaluateJsFunction(ctx, "minifyHTML()")
	if err != nil {
		return nil, nil, "", err
	}
	elementsSpec := &elementSpec.ElementSpec{}
	if err := json.Unmarshal([]byte(result), elementsSpec); err != nil {
		return nil, nil, "", fmt.Errorf(
			"failed to unmarshal ElementSpec json: %v, expected json, received: %s",
			err,
			result,
		)
	}

	result, _ = apPlugin.evaluateJsFunction(ctx, "mapElementsToJson()")
	idLocatorMap := &elementSpec.IdToLocatorMap{}
	if err := json.Unmarshal([]byte(result), idLocatorMap); err != nil {
		return nil, nil, "", fmt.Errorf(
			"failed to unmarshal IdToLocatorMap json: %v, expected json, received: %s",
			err,
			result,
		)
	}
	return elementsSpec, idLocatorMap, "css selector", nil
}

func (apPlugin *appiumPlugin) xmlMinification(ctx context.Context) (*elementSpec.ElementSpec, *elementSpec.IdToLocatorMap, locatr.SelectorType, error) {
	span := trace.SpanFromContext(ctx)

	pageSource, err := apPlugin.client.GetPageSource(ctx)
	if err != nil {
		return nil, nil, "", err
	}
	capabilities, err := apPlugin.client.GetCapabilities(ctx)
	if err != nil {
		return nil, nil, "", err
	}
	platFormName := capabilities.Value.PlatformName
	if platFormName == "" {
		platFormName = capabilities.Value.Cap.PlatformName
	}
	span.AddEvent("minifying source")
	eSpec, err := minifier.MinifyXMLSource(pageSource, platFormName)
	if err != nil {
		return nil, nil, "", err
	}
	span.AddEvent("mapping elements to json")
	locatrMap, err := minifier.MapXMLElementsToJson(pageSource, platFormName)
	if err != nil {
		return nil, nil, "", err
	}
	return eSpec, locatrMap, "xpath", nil
}

func (apPlugin *appiumPlugin) GetCurrentContext(ctx context.Context) string {
	ctx, span := tracing.StartSpan(ctx, "GetCurrentContext")
	defer span.End()

	capabilities, err := apPlugin.client.GetCapabilities(ctx)
	if err != nil {
		return ""
	}

	span.SetAttributes(attribute.String("platform", capabilities.Value.PlatformName))

	if strings.ToLower(capabilities.Value.PlatformName) != "andriod" {
		return ""
	}
	if currentActivity, err := apPlugin.client.GetCurrentActivity(ctx); err != nil {
		return currentActivity
	}
	return ""
}

func (apPlugin *appiumPlugin) IsValidLocator(ctx context.Context, locatr string) (bool, error) {
	ctx, span := tracing.StartSpan(ctx, "IsValidLocator")
	defer span.End()

	locator_type := "xpath"
	if apPlugin.client.IsWebView(ctx) {
		locator_type = "css selector"
	}

	span.SetAttributes(attribute.String("locator_type", locator_type))

	if err := apPlugin.client.FindElement(ctx, locatr, locator_type); err == nil {
		return true, nil
	} else {
		return false, err
	}
}

func (apLocatr *appiumLocatr) GetLocatrStr(ctx context.Context, userReq string) (*locatr.LocatrOutput, error) {
	ctx, span := tracing.StartSpan(ctx, "GetLocatrStr")
	defer span.End()

	locatrOutput, err := apLocatr.locatr.GetLocatorStr(ctx, userReq)
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
