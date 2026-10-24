package lsp

const (
	SymbolKindNamespace = 3
	SymbolKindString    = 15
)

// DocumentSymbolRequest is a textDocument/documentSymbol request.
type DocumentSymbolRequest struct {
	Request
	Params DocumentSymbolParams `json:"params"`
}

// DocumentSymbolParams identifies the document to inspect.
type DocumentSymbolParams struct {
	TextDocument TextDoucmentIdentifier `json:"textDocument"`
}

// DocumentSymbolResponse carries nested document symbols.
type DocumentSymbolResponse struct {
	Response
	Result []DocumentSymbol `json:"result"`
}

// DocumentSymbol describes a heading and its child headings.
type DocumentSymbol struct {
	Name           string           `json:"name"`
	Detail         string           `json:"detail,omitempty"`
	Kind           int              `json:"kind"`
	Range          Range            `json:"range"`
	SelectionRange Range            `json:"selectionRange"`
	Children       []DocumentSymbol `json:"children,omitempty"`
}
