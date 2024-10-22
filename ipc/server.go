package ipc

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"

	"github.com/vertexcover-io/locatr/locatr"
)

type locatorServer struct {
	locator      *ipcLocatr
	connections  map[net.Conn]chan *ipcResponse
	mutex        sync.RWMutex
	requestChan  chan any
	responseChan chan *ipcResponse
}

func (s *locatorServer) handleIncomingMessage(message string, conn net.Conn) error {
	var baseConn map[string]interface{}
	if err := json.Unmarshal([]byte(message), &baseConn); err != nil {
		return fmt.Errorf("Error unmarshaling message:", err)
	}

	connType, ok := baseConn["conn_type"].(string)
	if !ok {
		return fmt.Errorf("Error: conn_type not found or invalid")
	}

	switch connType {
	case string(clientRequestType):
		var req ipcRequest
		if err := json.Unmarshal([]byte(message), &req); err != nil {
			return fmt.Errorf("Error unmarshaling client request:", err)
		}
		resp := s.handleRequest(&req)
		s.mutex.RLock()
		respChan, ok := s.connections[conn]
		s.mutex.RUnlock()
		if ok {
			respChan <- resp
		}
	case string(clientResponseType):
		var resp ipcResponse
		if err := json.Unmarshal([]byte(message), &resp); err != nil {
			return fmt.Errorf("Error unmarshaling client response:", err)
		}
		s.responseChan <- &resp
	default:
		return fmt.Errorf("Unknown connection type:", connType)
	}
	return nil
}

func (s *locatorServer) handleRequest(req *ipcRequest) *ipcResponse {
	switch req.Method {
	case methodInit:
		return s.initLocator(req.Params)
	case methodGetLocator:
		return s.getLocator(req.Params)
	default:
		return makeErrorResponse(statusInvalidRequest, "Unknown method")
	}
}

func (s *locatorServer) initLocator(params map[string]any) *ipcResponse {
	configJSON, ok := params["config"].(map[string]any)
	if !ok {
		return makeErrorResponse(statusInvalidRequest, "Missing 'config' parameter")
	}

	configBytes, err := json.Marshal(configJSON)
	if err != nil {
		return makeErrorResponse(statusUnprocessableEntity, fmt.Sprintf("Failed to marshal config: %v", err))
	}
	conf := &locatr.LocatrConfig{}
	if err = json.Unmarshal(configBytes, conf); err != nil {
		return makeErrorResponse(statusUnprocessableEntity, fmt.Sprintf("Failed to unmarshal config: %v", err))
	}

	locator, err := NewIpcLocator(conf, s.requestChan, s.responseChan)
	if err != nil {
		return makeErrorResponse(statusInternalError, fmt.Sprintf("Failed to create locator: %v", err))
	}

	s.locator = locator
	return makeSuccessResponse(statusOk, "Locator initialized successfully")
}

func (s *locatorServer) getLocator(params map[string]any) *ipcResponse {
	userReq, ok := params["user_req"].(string)
	if !ok {
		return makeErrorResponse(statusInvalidRequest, "Missing 'user_req' parameter")
	}

	if s.locator == nil {
		return makeErrorResponse(statusInternalError, "Locator not initialized")
	}

	locator, err := s.locator.GetLocator(userReq)
	if err != nil {
		return makeErrorResponse(statusInternalError, err.Error())
	}
	return makeSuccessResponse(statusOk, map[string]string{"locator": locator})
}

func StartTCPServer(port int) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	defer listener.Close()

	fmt.Println("TCP server listening on port", port)

	server := &locatorServer{
		requestChan:  make(chan any, 1),
		responseChan: make(chan *ipcResponse, 1),
		connections:  make(map[net.Conn]chan *ipcResponse, 1),
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Error accepting connection: %v\n", err)
			continue
		}
		connResponseChan := make(chan *ipcResponse, 10)
		server.connections[conn] = connResponseChan
		go handleConnection(conn, server)
	}
}
