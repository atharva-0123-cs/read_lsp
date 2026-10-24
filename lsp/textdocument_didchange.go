package lsp

// TextDocumentDidChangeNotification is the didChange notification.
type TextDocumentDidChangeNotification struct {
	Notificaton
	Params DidChangeTextDocumentParams `json:"params"`
}

// DidChangeTextDocumentParams carries document changes.
type DidChangeTextDocumentParams struct {
	TextDocument   VersionTextDoucmentIdentifier    `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

// TextDocumentContentChangeEvent contains replacement document text.
type TextDocumentContentChangeEvent struct {
	Text string `json:"text"`
}
