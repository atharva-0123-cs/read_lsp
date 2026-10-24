package lsp

// HoverRequest is a textDocument/hover request.
type HoverRequest struct {
	Request
	Params HoverRequestParams `json:"params"`
}

// HoverRequestParams identifies the document and position to inspect.
type HoverRequestParams struct {
	TextDocumentPositionParams
}

// HoverResponse carries the result of a hover request.
type HoverResponse struct {
	Response
	Result HoverResult `json:"result"`
}

// HoverResult contains the markup returned for a hover request.
type HoverResult struct {
	Contents MarkupContent `json:"contents"`
}

// MarkupContent is Markdown or plain text returned by the server.
type MarkupContent struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}
