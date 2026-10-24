package lsp

const (
	SymbolKindNamespace = 3
	SymbolKindString    = 15
)

type DocumentSymbolRequest struct {
	Request
	Params DocumentSymbolParams `json:"params"`
}

type DocumentSymbolParams struct {
	TextDocument TextDoucmentIdentifier `json:"textDocument"`
}

type DocumentSymbolResponse struct {
	Response
	Result []DocumentSymbol `json:"result"`
}

type DocumentSymbol struct {
	Name           string           `json:"name"`
	Detail         string           `json:"detail,omitempty"`
	Kind           int              `json:"kind"`
	Range          Range            `json:"range"`
	SelectionRange Range            `json:"selectionRange"`
	Children       []DocumentSymbol `json:"children,omitempty"`
}
