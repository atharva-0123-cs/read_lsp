package lsp

// CompletionRequest is a textDocument/completion request.
type CompletionRequest struct {
	Request
	Params CompletionParams `json:"params"`
}

// CompletionParams identifies the document and cursor position.
type CompletionParams struct {
	TextDocumentPositionParams
}

// CompletionResponse carries the completion items.
type CompletionResponse struct {
	Response
	Result []CompletionItem `json:"result"`
}

// CompletionItem describes one completion suggestion.
type CompletionItem struct {
	Label            string    `json:"label"`
	Detail           string    `json:"detail"`
	Documentation    string    `json:"documentation"`
	InsertText       *string   `json:"insertText,omitempty"`
	InsertTextFormat int       `json:"insertTextFormat,omitempty"`
	TextEdit         *TextEdit `json:"textEdit,omitempty"`
}
