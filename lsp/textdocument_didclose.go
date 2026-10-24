package lsp

// DidCloseTextDocumentNotification is the didClose notification.
type DidCloseTextDocumentNotification struct {
	Notification
	Params DidCloseTextDocumentParams `json:"params"`
}

// DidCloseTextDocumentParams identifies the closed document.
type DidCloseTextDocumentParams struct {
	TextDocument TextDoucmentIdentifier `json:"textDocument"`
}
