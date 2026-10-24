package lsp

type FoldingRangeRequest struct {
	Request
	Params FoldingRangeParams `json:"params"`
}

type FoldingRangeParams struct {
	TextDocument TextDoucmentIdentifier `json:"textDocument"`
}

type FoldingRangeResponse struct {
	Response
	Result []FoldingRange `json:"result"`
}

type FoldingRange struct {
	StartLine      int    `json:"startLine"`
	StartCharacter *int   `json:"startCharacter,omitempty"`
	EndLine        int    `json:"endLine"`
	EndCharacter   *int   `json:"endCharacter,omitempty"`
	Kind           string `json:"kind,omitempty"`
}
