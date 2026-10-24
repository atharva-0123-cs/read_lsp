package lsp

import "encoding/json"

const JSONRPCVersion = "2.0"

type Request struct {
	RPC    string          `json:"jsonrpc"`
	ID     json.RawMessage `json:"id,omitempty"`
	Method string          `json:"method"`
}

type Response struct {
	RPC string          `json:"jsonrpc"`
	ID  json.RawMessage `json:"id,omitempty"`

	// Result
	// Error
}

type Notification struct {
	RPC    string `json:"jsonrpc"`
	Method string `json:"method"`
}

type Notificaton = Notification

type ResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Response
	Error ResponseError `json:"error"`
}

type NullResponse struct {
	Response
	Result any `json:"result"`
}

func NewResponse(id json.RawMessage) Response {
	return Response{
		RPC: JSONRPCVersion,
		ID:  id,
	}
}

func NewNotification(method string) Notification {
	return Notification{
		RPC:    JSONRPCVersion,
		Method: method,
	}
}

func NewErrorResponse(id json.RawMessage, code int, message string) ErrorResponse {
	return ErrorResponse{
		Response: NewResponse(id),
		Error: ResponseError{
			Code:    code,
			Message: message,
		},
	}
}

func NewNullResponse(id json.RawMessage) NullResponse {
	return NullResponse{
		Response: NewResponse(id),
		Result:   nil,
	}
}
