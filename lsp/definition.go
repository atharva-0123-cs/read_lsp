package lsp

// DefinitionRequest is a textDocument/definition request.
type DefinitionRequest struct {
	Request
	Params DefinitionParams `json:"params"`
}

// DefinitionParams identifies the document and position to resolve.
type DefinitionParams struct {
	TextDocumentPositionParams
}

// DefinitionResponse carries the definition location, if any.
type DefinitionResponse struct {
	Response
	Result *Location `json:"result"`
}
