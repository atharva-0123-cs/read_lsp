package lsp

type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageid"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

type TextDoucmentIdentifier struct {
	URI string `json:"uri"`
}

type VersionTextDoucmentIdentifier struct {
	TextDoucmentIdentifier
	Version int `json:"version"`
}

type TextDocumentPositionParams struct {
	TextDocument TextDoucmentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type WorkspaceEdit struct {
	Changes map[string][]TextEdit `json:"changes"`
}

type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}
