package lsp

// DidOpenTextDocumentNotifiction is the didOpen notification.
type DidOpenTextDocumentNotifiction struct {
	Notificaton
	Params DidOpenTextDocumentParams `json:"params"`
}

// DidOpenTextDocumentParams carries the opened document.
type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}
