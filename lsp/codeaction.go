package lsp

// CodeActionRequest is a textDocument/codeAction request.
type CodeActionRequest struct {
	Request
	Params TextDocumentCodeActionParams `json:"params"`
}

// TextDocumentCodeActionParams identifies the document and range.
type TextDocumentCodeActionParams struct {
	TextDocument TextDoucmentIdentifier `json:"textDocument"`
	Range        Range                  `json:"range"`
	Context      CodeActionContext      `json:"context"`
}

// TextDocumentCodeActionResponse carries quick fixes.
type TextDocumentCodeActionResponse struct {
	Response
	Result []CodeAction `json:"result"`
}

// CodeActionContext carries client-specific action context.
type CodeActionContext struct {
	// Add fields for CodeActionContext as needed
}

// CodeAction is an edit or command offered as a quick fix.
type CodeAction struct {
	Title   string         `json:"title"`
	Edit    *WorkspaceEdit `json:"edit,omitempty"`
	Command *Command       `json:"command,omitempty"`
}

// Command describes a server command that can be executed by the client.
type Command struct {
	Title     string        `json:"title"`
	Command   string        `json:"command"`
	Arguments []interface{} `json:"arguments,omitempty"`
}
