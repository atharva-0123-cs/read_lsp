package lsp

type ReferenceRequest struct {
	Request
	Params ReferenceParams `json:"params"`
}

type ReferenceParams struct {
	TextDocumentPositionParams
	Context ReferenceContext `json:"context"`
}

type ReferenceContext struct {
	IncludeDeclaration bool `json:"includeDeclaration"`
}

type ReferenceResponse struct {
	Response
	Result []Location `json:"result"`
}
