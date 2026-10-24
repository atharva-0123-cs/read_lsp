package lsp

import "encoding/json"

type CancelRequestNotification struct {
	Notification
	Params CancelParams `json:"params"`
}

type CancelParams struct {
	ID json.RawMessage `json:"id"`
}
