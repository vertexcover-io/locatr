package seleniumLocatr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	locatr "github.com/vertexcover-io/locatr/golang"
	"github.com/vertexcover-io/locatr/golang/elementSpec"
	"github.com/vertexcover-io/locatr/golang/tracing"
	"github.com/vertexcover-io/selenium"
	"go.opentelemetry.io/otel/trace"
)

type seleniumPlugin struct {
	driver selenium.WebDriver
}

type seleniumLocatr struct {
	driver selenium.WebDriver
	locatr *locatr.BaseLocatr
}

var ErrUnableToLoadJsScriptSelenium = errors.New("unable to load js script through selenium")

// NewRemoteConnSeleniumLocatr Create a new selenium locatr with selenium session.
func NewRemoteConnSeleniumLocatr(
	ctx context.Context,
	serverUrl string,
	sessionId string,
	opt locatr.BaseLocatrOptions,
) (*seleniumLocatr, error) {
	span := trace.SpanFromContext(ctx)

	span.AddEvent("Connecting to remote selenium")

	wd, err := selenium.ConnectRemote(serverUrl, sessionId)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to remote selenium instance: %w", err)
	}

	span.AddEvent("Connection to remote selenium established")

	seleniumPlugin := seleniumPlugin{driver: wd}
	locatr := seleniumLocatr{
		driver: wd,
		locatr: locatr.NewBaseLocatr(&seleniumPlugin, opt),
	}
	return &locatr, nil
}

func NewSeleniumLocatr(
	ctx context.Context,
	driver selenium.WebDriver,
	options locatr.BaseLocatrOptions,
) (*seleniumLocatr, error) {
	span := trace.SpanFromContext(ctx)

	span.AddEvent("Connecting to selenium driver")
	plugin := &seleniumPlugin{
		driver: driver,
	}
	baseLocatr := locatr.NewBaseLocatr(plugin, options)
	return &seleniumLocatr{
		driver: driver,
		locatr: baseLocatr,
	}, nil

}

// Close close the selenium session.
func (sl *seleniumLocatr) Close() error {
	return sl.driver.Quit()
}

func (sl *seleniumPlugin) evaluateJsFunction(ctx context.Context, function string) (string, error) {
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

func (sl *seleniumPlugin) evaluateJsScript(ctx context.Context, script string) error {
	_, err := sl.driver.ExecuteScript(script, nil)
	if err != nil {
		return fmt.Errorf("failed to evaluate JS script: %w", err)
	}
	return nil
}

func (sl *seleniumLocatr) GetLocatrStr(ctx context.Context, userReq string) (*locatr.LocatrOutput, error) {
	ctx, span := tracing.StartSpan(ctx, "GetLocatrStr")
	defer span.End()

	locatorOutput, err := sl.locatr.GetLocatorStr(ctx, userReq)
	if err != nil {
		return nil, fmt.Errorf("error getting locator string: %w", err)
	}
	return locatorOutput, nil
}

func (pl *seleniumLocatr) WriteResultsToFile() {
	pl.locatr.WriteLocatrResultsToFile()
}

func (pl *seleniumLocatr) GetLocatrResults() []locatr.LocatrResult {
	return pl.locatr.GetLocatrResults()
}

func (sl *seleniumPlugin) GetMinifiedDomAndLocatorMap(ctx context.Context) (
	*elementSpec.ElementSpec,
	*elementSpec.IdToLocatorMap,
	locatr.SelectorType,
	error,
) {
	ctx, span := tracing.StartSpan(ctx, "GetMinifiedDomAndLocatorMap")
	defer span.End()

	span.AddEvent("injecting HTML minifer script")
	if err := sl.evaluateJsScript(ctx, locatr.HTML_MINIFIER_JS_CONTENT); err != nil {
		return nil, nil, "", fmt.Errorf("%v : %v", ErrUnableToLoadJsScriptSelenium, err)
	}

	span.AddEvent("evaluating minifyHTML function")
	result, err := sl.evaluateJsFunction(ctx, "minifyHTML()")
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

	span.AddEvent("evaluating mapElementsToJson function")
	result, _ = sl.evaluateJsFunction(ctx, "mapElementsToJson()")
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

func (sl *seleniumPlugin) GetCurrentContext(ctx context.Context) string {
	ctx, span := tracing.StartSpan(ctx, "GetCurrentContext")
	defer span.End()

	span.AddEvent("fetching current window location")
	if value, err := sl.evaluateJsFunction(ctx, "window.location.href"); err == nil {
		return value
	} else {
		return ""
	}
}

func (sl *seleniumPlugin) IsValidLocator(ctx context.Context, locatrString string) (bool, error) {
	ctx, span := tracing.StartSpan(ctx, "IsValidLocator")
	defer span.End()

	span.AddEvent("injecting HTML minifier script")
	if err := sl.evaluateJsScript(ctx, locatr.HTML_MINIFIER_JS_CONTENT); err != nil {
		return false, fmt.Errorf("%v : %v", ErrUnableToLoadJsScriptSelenium, err)
	}

	span.AddEvent("evaluating isValidLocator function")
	value, err := sl.evaluateJsFunction(ctx, fmt.Sprintf("isValidLocator('%s')", locatrString))
	if value == "true" && err == nil {
		return true, nil
	} else {
		return false, err
	}
}
