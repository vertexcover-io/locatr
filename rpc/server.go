package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	"github.com/vertexcover-io/locatr/plugins"
	pb "github.com/vertexcover-io/locatr/rpc/grpc"
	"google.golang.org/grpc"
)

type locatorServer struct {
	pb.UnimplementedIpcServiceServer
	locator *plugins.GrpcLocatr
}

func (s *locatorServer) InitLocator(ctx context.Context, req *pb.InitLocatorRequest) (*pb.InitLocatorResponse, error) {
	conf := &plugins.GrpcLocatorConfig{}
	err := json.Unmarshal(req.Config, conf)
	if err != nil {
		return &pb.InitLocatorResponse{Success: false, Error: fmt.Sprintf("Failed to unmarshal config: %v", err)}, nil
	}

	locator, err := plugins.NewRpcLocator(conf)
	if err != nil {
		return &pb.InitLocatorResponse{Success: false, Error: fmt.Sprintf("Failed to create locator: %v", err)}, nil
	}

	s.locator = locator
	return &pb.InitLocatorResponse{Success: true}, nil
}

func (s *locatorServer) GetLocator(ctx context.Context, req *pb.GetLocatorRequest) (*pb.GetLocatorResponse, error) {
	locator, err := s.locator.GetLocator(req.UserReq)
	if err != nil {
		return &pb.GetLocatorResponse{Error: err.Error()}, nil
	}
	return &pb.GetLocatorResponse{LocatorStr: locator}, nil
}

func StartGRPCServer(port int) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterIpcServiceServer(s, &locatorServer{})
	fmt.Println("gRPC server listening on port", port)

	if err := s.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %v", err)
	}
	return nil
}
