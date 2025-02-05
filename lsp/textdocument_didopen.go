package lsp

type DidOpenTextDocumentNotifiction struct {
	Notificaton
	Params DidOpenTextDocumentParams `json:"params"`
}

type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}
