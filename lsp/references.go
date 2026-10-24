package lsp

// ReferenceRequest is a textDocument/references request.
type ReferenceRequest struct {
	Request
	Params ReferenceParams `json:"params"`
}

// ReferenceParams identifies a position and reference options.
type ReferenceParams struct {
	TextDocumentPositionParams
	Context ReferenceContext `json:"context"`
}

// ReferenceContext controls whether declarations are included.
type ReferenceContext struct {
	IncludeDeclaration bool `json:"includeDeclaration"`
}

// ReferenceResponse carries locations that reference the target.
type ReferenceResponse struct {
	Response
	Result []Location `json:"result"`
}
