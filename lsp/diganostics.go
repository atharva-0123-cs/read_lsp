package lsp

// PublishDiagnosticsNotification is the publishDiagnostics notification.
type PublishDiagnosticsNotification struct {
	Notificaton
	Params PublishDiagnosticsParams `json:"params"`
}

// PublishDiagnosticsParams carries diagnostics for one document.
type PublishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// Diagnostic describes one Markdown problem in a document.
type Diagnostic struct {
	Range    Range  `json:"range"`
	Severity int    `json:"severity"`
	Code     string `json:"code,omitempty"`
	Source   string `json:"source"`
	Message  string `json:"message"`
}
