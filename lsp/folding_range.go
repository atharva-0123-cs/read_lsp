package lsp

// FoldingRangeRequest is a textDocument/foldingRange request.
type FoldingRangeRequest struct {
	Request
	Params FoldingRangeParams `json:"params"`
}

// FoldingRangeParams identifies the document to fold.
type FoldingRangeParams struct {
	TextDocument TextDoucmentIdentifier `json:"textDocument"`
}

// FoldingRangeResponse carries the document folding ranges.
type FoldingRangeResponse struct {
	Response
	Result []FoldingRange `json:"result"`
}

// FoldingRange describes a foldable line range.
type FoldingRange struct {
	StartLine      int    `json:"startLine"`
	StartCharacter *int   `json:"startCharacter,omitempty"`
	EndLine        int    `json:"endLine"`
	EndCharacter   *int   `json:"endCharacter,omitempty"`
	Kind           string `json:"kind,omitempty"`
}
