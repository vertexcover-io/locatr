package main

//go:generate mkdir -p rpc
//go:generate protoc --go_out=rpc/ --go-grpc_out=rpc/ ipc.proto
