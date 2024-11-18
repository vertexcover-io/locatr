package locatr

import (
	"context"
	"fmt"
	"github.com/mafredri/cdp"
	"github.com/mafredri/cdp/protocol/runtime"
	"github.com/mafredri/cdp/rpcc"
	"log"
)

var ConnectionsSlice = make([]*rpcc.Conn, 0)

func CreateRpccConnection(port int, pageId string) (int, error) {
	ctx := context.Background()
	wsUrl := fmt.Sprintf("ws://localhost:%d/devtools/page/%s", port, (pageId))
	conn, err := rpcc.DialContext(ctx, wsUrl, rpcc.WithWriteBufferSize(1048576))
	if err != nil {
		log.Println("could not connect to %s: %v", wsUrl, err)
		return -1, err
	}
	ConnectionsSlice = append(ConnectionsSlice, conn)
	return len(ConnectionsSlice) - 1, nil
}

func CloseRpccConnection(index int) error {
	if index < 0 || index >= len(ConnectionsSlice) {
		return fmt.Errorf("Invalid index to close connection %d", index)
	}
	ConnectionsSlice[index].Close()
	ConnectionsSlice[index] = nil
	return nil
}

func ExecuteJs(connectionId int, jsString string) string {
	fmt.Println("The js string is ", len(jsString), " long and The connection id is ", connectionId)
	if connectionId < 0 || connectionId >= (len(ConnectionsSlice)) {
		return ("Connection id is invalid")
	}
	conn := ConnectionsSlice[(connectionId)]
	c := cdp.NewClient(conn)
	if c == nil {
		return "Web Socket Client not found"
	}
	pageRuntime := c.Runtime
	result, err := pageRuntime.Evaluate(context.Background(), &runtime.EvaluateArgs{
		Expression: jsString,
	})
	if err != nil {
		log.Fatal("error is here", err)
		return string(err.Error())
	}
	return string(result.Result.Value)
}
