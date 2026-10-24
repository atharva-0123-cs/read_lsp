package lsp

type DidCloseTextDocumentNotification struct {
	Notification
	Params DidCloseTextDocumentParams `json:"params"`
}

type DidCloseTextDocumentParams struct {
	TextDocument TextDoucmentIdentifier `json:"textDocument"`
}
