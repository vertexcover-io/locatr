package locatr

import (
	"context"
	"fmt"
	"strconv"

	"github.com/mafredri/cdp"
	"github.com/mafredri/cdp/protocol/runtime"
	"github.com/mafredri/cdp/rpcc"
	"gopkg.in/validator.v2"
)

type cdpPlugin struct {
	client *cdp.Client
	PluginInterface
}

type cdpLocatr struct {
	client     *cdp.Client
	connection *rpcc.Conn
	locatr     *BaseLocatr
	LocatrInterface
}

type CdpConnectionOptions struct {
	HostName string
	Port     int `binding:"required"`
}

func CreateCdpConnection(options CdpConnectionOptions) (*rpcc.Conn, error) {
	ctx := context.Background()
	if len(options.HostName) == 0 {
		options.HostName = "localhost"
	}
	optionValidator := validator.NewValidator()
	if err := optionValidator.Validate(options); err != nil {
		return nil, err
	}
	wsUrl, err := getWebsocketDebugUrl(fmt.Sprintf("http://%s:%d", options.HostName, options.Port), 0, ctx)
	if err != nil {
		return nil, err
	}
	conn, err := rpcc.DialContext(ctx, wsUrl, rpcc.WithWriteBufferSize(1048576))
	if err != nil {
		return nil, fmt.Errorf("Could not connect to cdp server: %s, err: %w", wsUrl, err)
	}
	return conn, nil
}

func NewCdpLocatr(connection *rpcc.Conn, locatrOptions BaseLocatrOptions) (*cdpLocatr, error) {
	client := cdp.NewClient(connection)
	cdpPlugin := &cdpPlugin{client: client}
	return &cdpLocatr{
		client:     client,
		locatr:     NewBaseLocatr(cdpPlugin, locatrOptions),
		connection: connection,
	}, nil
}

func (cdpPlugin *cdpPlugin) evaluateJsFunction(function string) (string, error) {
	pageRuntime := cdpPlugin.client.Runtime
	result, err := pageRuntime.Evaluate(context.Background(), &runtime.EvaluateArgs{
		Expression: function,
	})
	if err != nil {
		return "", fmt.Errorf("Error evaluating js function with cdp: %w", err)
	}
	// remove quotation, escape chracters from the string to unmarshal the json later.
	resultString := string(result.Result.Value)
	str, err := strconv.Unquote(resultString)
	if err != nil {
		return resultString, nil
	}
	return str, err

}

func (cdpPlugin *cdpPlugin) evaluateJsScript(scriptContent string) error {
	pageRuntime := cdpPlugin.client.Runtime
	_, err := pageRuntime.Evaluate(context.Background(), &runtime.EvaluateArgs{
		Expression: scriptContent,
	})
	if err != nil {
		return fmt.Errorf("Error evaluating js script with cdp: %w", err)
	}
	return nil
}

func (cdpLocatr *cdpLocatr) GetLocatrStr(userReq string) (string, error) {
	locatrStr, err := cdpLocatr.locatr.getLocatorStr(userReq)
	if err != nil {
		return "", fmt.Errorf("error getting locator string: %w", err)
	}
	return locatrStr, nil

}
func (cdpLocatr *cdpLocatr) WriteResultsToFile() {
	cdpLocatr.locatr.writeLocatrResultsToFile()
}

func (cdpLocatr *cdpLocatr) GetLocatrResults() []locatrResult {
	return cdpLocatr.locatr.getLocatrResults()
}
