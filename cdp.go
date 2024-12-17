package locatr

import (
	"context"
	"fmt"
	"strconv"

	"github.com/mafredri/cdp"
	"github.com/mafredri/cdp/protocol/runtime"
	"github.com/mafredri/cdp/rpcc"
)

type cdpPlugin struct {
	client *cdp.Client
	PluginInterface
}

type cdpLocatr struct {
	client     *cdp.Client
	connection *rpcc.Conn
	locatr     *BaseLocatr
}

type CdpConnectionOptions struct {
	Host   string
	Port   int    `validate:"required"`
	PageId string `validate:"required"`
}

func CreateCdpConnection(options CdpConnectionOptions) (*rpcc.Conn, error) {
	if len(options.Host) == 0 {
		options.Host = "localhost"
	}
	ctx := context.Background()
	wsUrl := fmt.Sprintf("ws://%s:%d/devtools/page/%s", options.Host, options.Port, (options.PageId))
	conn, err := rpcc.DialContext(ctx, wsUrl, rpcc.WithWriteBufferSize(1048576))
	if err != nil {
		return nil, fmt.Errorf("Could not connect to cdp server: %s, err: %v", wsUrl, err)
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

func (cdpLocatr *cdpLocatr) CloseCdpConnection() {
	cdpLocatr.connection.Close()
}

func (cdpPlugin *cdpPlugin) evaluateJsFunction(function string) (string, error) {
	pageRuntime := cdpPlugin.client.Runtime
	result, err := pageRuntime.Evaluate(context.Background(), &runtime.EvaluateArgs{
		Expression: function,
	})
	if err != nil {
		return "", fmt.Errorf("Error evaluating js function with cdp: %v", err)
	}
	// remove quotation, escape chracters from the string to unmarshal the json later.
	resultString := string(result.Result.Value)
	str, err := strconv.Unquote(resultString)
	if err != nil {
		return resultString, err
	}
	return str, err

}

func (cdpPlugin *cdpPlugin) evaluateJsScript(scriptContent string) error {
	pageRuntime := cdpPlugin.client.Runtime
	_, err := pageRuntime.Evaluate(context.Background(), &runtime.EvaluateArgs{
		Expression: scriptContent,
	})
	if err != nil {
		return fmt.Errorf("Error evaluating js script with cdp: %v", err)
	}
	return nil
}

func (cdpLocatr *cdpLocatr) GetLocatr(userReq string) (string, error) {
	locatrStr, err := cdpLocatr.locatr.getLocatorStr(userReq)
	if err != nil {
		return "", fmt.Errorf("error getting locator string: %v", err)
	}
	return locatrStr, nil

}
func (cdpLocatr *cdpLocatr) WriteResultsToFile() {
	cdpLocatr.locatr.writeLocatrResultsToFile()
}

func (cdpLocatr *cdpLocatr) GetLocatrResults() []locatrResult {
	return cdpLocatr.locatr.getLocatrResults()
}
