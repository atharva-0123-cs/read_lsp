package lsp

import "encoding/json"

const JSONRPCVersion = "2.0"

// Request is the common JSON-RPC request envelope.
type Request struct {
	RPC    string          `json:"jsonrpc"`
	ID     json.RawMessage `json:"id,omitempty"`
	Method string          `json:"method"`
}

// Response is the common JSON-RPC response envelope.
type Response struct {
	RPC string          `json:"jsonrpc"`
	ID  json.RawMessage `json:"id,omitempty"`

	// Result
	// Error
}

// Notification is the common JSON-RPC notification envelope.
type Notification struct {
	RPC    string `json:"jsonrpc"`
	Method string `json:"method"`
}

// Notificaton is an alias for Notification.
type Notificaton = Notification

// ResponseError describes a JSON-RPC error object.
type ResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ErrorResponse is a JSON-RPC response containing an error.
type ErrorResponse struct {
	Response
	Error ResponseError `json:"error"`
}

// NullResponse is a successful response with a null result.
type NullResponse struct {
	Response
	Result any `json:"result"`
}

// NewResponse creates a response with the given request ID.
func NewResponse(id json.RawMessage) Response {
	return Response{
		RPC: JSONRPCVersion,
		ID:  id,
	}
}

// NewNotification creates a notification with the given method.
func NewNotification(method string) Notification {
	return Notification{
		RPC:    JSONRPCVersion,
		Method: method,
	}
}

// NewErrorResponse creates an error response.
func NewErrorResponse(id json.RawMessage, code int, message string) ErrorResponse {
	return ErrorResponse{
		Response: NewResponse(id),
		Error: ResponseError{
			Code:    code,
			Message: message,
		},
	}
}

// NewNullResponse creates a successful null response.
func NewNullResponse(id json.RawMessage) NullResponse {
	return NullResponse{
		Response: NewResponse(id),
		Result:   nil,
	}
}
