package lsp

import "encoding/json"

// CancelRequestNotification is the $/cancelRequest notification.
type CancelRequestNotification struct {
	Notification
	Params CancelParams `json:"params"`
}

// CancelParams carries the ID of the request to cancel.
type CancelParams struct {
	ID json.RawMessage `json:"id"`
}
