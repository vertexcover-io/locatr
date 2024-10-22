package ipc

type method string
type statusCode int
type params map[string]any
type connType string

const (
	// serverRequestType is the type of request
	serverRequestType connType = "server_request"
	// serverResponseType is the type of response
	serverResponseType = "server_response"
	// clientRequestType is the type of request
	clientRequestType = "client_request"
	// clientResponseType is the type of response
	clientResponseType = "client_response"
)

const (
	// methodInit is the method to initialize the locator
	methodInit method = "init"
	// methodGetLocator is the method to get the locator
	methodGetLocator = "get_locator"
	// methodEvaluateJs is method used to evaluate_js in client
	methodEvaluateJs = "evaluate_js"
)

const (
	// statusOk is the status code for a successful response
	statusOk statusCode = 200
	// statusInvalidRequest is the status code for an invalid request
	statusInvalidRequest = 400
	// statusNotFound is the status code for a not found response
	statusNotFound = 404
	// statusUnprocessableEntity is the status code for an unprocessable entity
	statusUnprocessableEntity = 422
	// statusInternalError is the status code for an error response
	statusInternalError = 500
)

type ipcResponse struct {
	ConnType   connType   `json:"conn_type"`
	StatusCode statusCode `json:"status_code"`
	Data       any        `json:"data"`
	Error      string     `json:"error"`
}

type ipcRequest struct {
	ConnType connType `json:"conn_type"`
	Method   method   `json:"method"`
	Params   params   `json:"params"`
}
