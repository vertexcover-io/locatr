package cdpLocatr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/mafredri/cdp"
	"github.com/mafredri/cdp/devtool"
	"github.com/mafredri/cdp/protocol/runtime"
	"github.com/mafredri/cdp/rpcc"
	locatr "github.com/vertexcover-io/locatr/golang"
	"github.com/vertexcover-io/locatr/golang/elementSpec"
	"github.com/vertexcover-io/locatr/golang/tracing"
	"go.opentelemetry.io/otel/trace"
	"gopkg.in/validator.v2"
)

type cdpPlugin struct {
	client *cdp.Client
}

type cdpLocatr struct {
	client     *cdp.Client
	connection *rpcc.Conn
	locatr     *locatr.BaseLocatr
}

type CdpConnectionOptions struct {
	HostName string
	Port     int `binding:"required"`
}

var ErrUnableToLoadJsScriptsThroughCdp = errors.New("unable to load js script through cdp")

func CreateCdpConnection(ctx context.Context, options CdpConnectionOptions) (*rpcc.Conn, error) {
	span := trace.SpanFromContext(ctx)

	if len(options.HostName) == 0 {
		options.HostName = "localhost"
	}
	optionValidator := validator.NewValidator()
	if err := optionValidator.Validate(options); err != nil {
		return nil, err
	}
	url := fmt.Sprintf("http://%s:%d", options.HostName, options.Port)
	wsUrl, err := getWebsocketDebugUrl(ctx, url, 0)
	if err != nil {
		return nil, err
	}

	span.AddEvent("Dailing cdp")

	conn, err := rpcc.DialContext(ctx, wsUrl, rpcc.WithWriteBufferSize(1048576))
	if err != nil {
		return nil, fmt.Errorf("could not connect to cdp server: %s, err: %w", wsUrl, err)
	}

	span.AddEvent("CDP connected")
	return conn, nil
}

func NewCdpLocatr(connection *rpcc.Conn, locatrOptions locatr.BaseLocatrOptions) (*cdpLocatr, error) {
	client := cdp.NewClient(connection)
	cdpPlugin := &cdpPlugin{client: client}
	return &cdpLocatr{
		client:     client,
		locatr:     locatr.NewBaseLocatr(cdpPlugin, locatrOptions),
		connection: connection,
	}, nil
}

func (cPlugin *cdpPlugin) GetMinifiedDomAndLocatorMap(ctx context.Context) (
	*elementSpec.ElementSpec,
	*elementSpec.IdToLocatorMap,
	locatr.SelectorType,
	error,
) {
	ctx, span := tracing.StartSpan(ctx, "GetMinifiedDomAndLocatorMap")
	defer span.End()

	span.AddEvent("inject HTML minifier script")
	if err := cPlugin.evaluateJsScript(ctx, locatr.HTML_MINIFIER_JS_CONTENT); err != nil {
		return nil, nil, "", fmt.Errorf("%v : %v", ErrUnableToLoadJsScriptsThroughCdp, err)
	}

	span.AddEvent("evaluate minifyHTML function")
	result, err := cPlugin.evaluateJsFunction(ctx, "minifyHTML()")
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

	span.AddEvent("evaluate mapElementsToJson")
	result, _ = cPlugin.evaluateJsFunction(ctx, "mapElementsToJson()")
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

func (cdpPlugin *cdpPlugin) evaluateJsFunction(ctx context.Context, function string) (string, error) {
	pageRuntime := cdpPlugin.client.Runtime
	result, err := pageRuntime.Evaluate(ctx, &runtime.EvaluateArgs{
		Expression: function,
	})
	if err != nil {
		return "", fmt.Errorf("error evaluating js function with cdp: %w", err)
	}
	// remove quotation, escape characters from the string to unmarshal the json later.
	resultString := string(result.Result.Value)
	str, err := strconv.Unquote(resultString)
	if err != nil {
		return resultString, nil
	}
	return str, err

}

func (cdpPlugin *cdpPlugin) evaluateJsScript(ctx context.Context, scriptContent string) error {
	pageRuntime := cdpPlugin.client.Runtime
	_, err := pageRuntime.Evaluate(ctx, &runtime.EvaluateArgs{
		Expression: scriptContent,
	})
	if err != nil {
		return fmt.Errorf("error evaluating js script with cdp: %w", err)
	}
	return nil
}

func (cdpLocatr *cdpLocatr) GetLocatrStr(ctx context.Context, userReq string) (*locatr.LocatrOutput, error) {
	ctx, span := tracing.StartSpan(ctx, "GetLocatrStr")
	defer span.End()

	locatrOutput, err := cdpLocatr.locatr.GetLocatorStr(ctx, userReq)
	if err != nil {
		return nil, fmt.Errorf("error getting locator string: %w", err)
	}
	return locatrOutput, nil

}
func (cdpLocatr *cdpLocatr) WriteResultsToFile() {
	cdpLocatr.locatr.WriteLocatrResultsToFile()
}

func (cdpLocatr *cdpLocatr) GetLocatrResults() []locatr.LocatrResult {
	return cdpLocatr.locatr.GetLocatrResults()
}

func filterTargets(pages []*devtool.Target) []*devtool.Target {
	newTargets := []*devtool.Target{}
	for _, target := range pages {
		if target.Type == "page" && !strings.HasPrefix("DevTools", target.Title) {
			newTargets = append(newTargets, target)
		}
	}
	return newTargets
}

func getWebsocketDebugUrl(ctx context.Context, url string, tabIndex int) (string, error) {
	devt := devtool.New(url)
	targets, err := devt.List(ctx)
	if err != nil {
		return "", err
	}
	targetsFiltered := filterTargets(targets)
	for indx, target := range targetsFiltered {
		if indx == tabIndex {
			return target.WebSocketDebuggerURL, nil
		}
	}
	return "", fmt.Errorf("tab with index %d not present in the browser", tabIndex)
}

func (cPlugin *cdpPlugin) GetCurrentContext(ctx context.Context) string {
	ctx, span := tracing.StartSpan(ctx, "GetCurrentContext")
	defer span.End()

	span.AddEvent("fetching window location")
	if value, err := cPlugin.evaluateJsFunction(ctx, "window.location.href"); err == nil {
		return value
	} else {
		return ""
	}
}

func (cPlugin *cdpPlugin) IsValidLocator(ctx context.Context, locatrString string) (bool, error) {
	ctx, span := tracing.StartSpan(ctx, "IsValidLocator")
	defer span.End()

	span.AddEvent("injecting HTML minifier script")
	if err := cPlugin.evaluateJsScript(ctx, locatr.HTML_MINIFIER_JS_CONTENT); err != nil {
		return false, fmt.Errorf("%v : %v", ErrUnableToLoadJsScriptsThroughCdp, err)
	}
	span.AddEvent("evaluating valid locator check")
	value, err := cPlugin.evaluateJsFunction(ctx, fmt.Sprintf("isValidLocator('%s')", locatrString))
	if value == "true" && err == nil {
		return true, nil
	} else {
		return false, err
	}
}
