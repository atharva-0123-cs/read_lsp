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
