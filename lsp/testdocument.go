package lsp

// TextDocumentItem describes an opened text document.
type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

// TextDoucmentIdentifier identifies a text document by URI.
type TextDoucmentIdentifier struct {
	URI string `json:"uri"`
}

// VersionTextDoucmentIdentifier adds a version to a document identifier.
type VersionTextDoucmentIdentifier struct {
	TextDoucmentIdentifier
	Version int `json:"version"`
}

// TextDocumentPositionParams identifies a document and cursor position.
type TextDocumentPositionParams struct {
	TextDocument TextDoucmentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// Position is a zero-based line and character position.
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Location is a range within a document identified by URI.
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// Range is a start and end position within a document.
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// WorkspaceEdit groups text edits by document URI.
type WorkspaceEdit struct {
	Changes map[string][]TextEdit `json:"changes"`
}

// TextEdit replaces a range with new text.
type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}
