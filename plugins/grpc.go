package plugins

import (
	"context"
	"fmt"

	"github.com/vertexcover-io/locatr/locatr"
	pb "github.com/vertexcover-io/locatr/rpc/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type rpcPlugin struct {
	locatr.PluginInterface
	pythonClient pb.IpcServiceClient
}

type GrpcLocatr struct {
	locatr *locatr.BaseLocatr
	plugin *playwrightPlugin
}

type GrpcLocatorConfig struct {
	locatr.LocatrConfig
	ConnTarget string `json:"conn_target"`
}

func NewRpcLocator(conf *GrpcLocatorConfig) (*GrpcLocatr, error) {
	conn, err := grpc.NewClient(conf.ConnTarget, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Python server: %v", err)
	}

	pythonClient := pb.NewIpcServiceClient(conn)
	rpcPlugin := &rpcPlugin{
		pythonClient: pythonClient,
	}

	baseLocatr, err := locatr.NewBaseLocatr(rpcPlugin, &locatr.LocatrConfig{
		LlmConfig: locatr.LlmConfig{
			ApiKey:   conf.LlmConfig.ApiKey,
			Provider: conf.LlmConfig.Provider,
			Model:    conf.LlmConfig.Model,
		},
		CachePath: conf.CachePath,
	})
	if err != nil {
		return nil, err
	}

	return &GrpcLocatr{
		locatr: baseLocatr,
	}, nil
}

func (rp *rpcPlugin) EvaluateJs(jsStr string) string {
	ctx := context.Background()
	response, err := rp.pythonClient.EvaluateJs(ctx, &pb.EvaluateJsRequest{JsStr: jsStr})
	if err != nil {
		return fmt.Sprintf("Error evaluating JS: %v", err)
	}
	return response.Result
}

func (rl *GrpcLocatr) GetLocator(userReq string) (string, error) {
	return rl.locatr.GetLocatorStr(userReq)
}
