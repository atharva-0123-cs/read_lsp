package lsp

type HoverRequest struct {
	Request
	Params HoverRequestParams `json:"params"`
}

type HoverRequestParams struct {
	TextDocumentPositionParams
}

type HoverResponse struct {
	Response
	Result HoverResult `json:"result"`
}

type HoverResult struct {
	Contents string `json:"contents"`
}
