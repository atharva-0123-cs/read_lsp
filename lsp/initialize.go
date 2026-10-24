package lsp

import "encoding/json"

// InitializeRequest is the LSP initialize request.
type InitializeRequest struct {
	Request
	Params InitializeRequestParams `json:"params"`
}

// InitializeRequestParams carries client information.
type InitializeRequestParams struct {
	ClientInfo *ClientInfo `json:"clientInfo"`

	// ... there's tons more goes here
}

// ClientInfo identifies the connected editor.
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// InitializeResponse carries server capabilities and metadata.
type InitializeResponse struct {
	Response
	Result InitializeResult `json:"result"`
}

// InitializeResult is the initialize result payload.
type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   ServerInfo         `json:"serverInfo"`
}

// ServerCapabilities advertises the features supported by the server.
type ServerCapabilities struct {
	TextDocumentSync TextDocumentSyncOptions `json:"textDocumentSync"`

	HoverProvider          bool           `json:"hoverProvider"`
	DefinitionProvider     bool           `json:"definitionProvider"`
	ReferencesProvider     bool           `json:"referencesProvider"`
	DocumentSymbolProvider bool           `json:"documentSymbolProvider"`
	FoldingRangeProvider   bool           `json:"foldingRangeProvider"`
	CodeActionProvider     bool           `json:"codeActionProvider"`
	CompletionProvider     map[string]any `json:"completionProvider"`
}

// TextDocumentSyncOptions controls document synchronization.
type TextDocumentSyncOptions struct {
	OpenClose bool `json:"openClose"`
	Change    int  `json:"change"`
}

// ServerInfo identifies the language server.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// NewInitializeResponse creates an initialize response with an integer ID.
func NewInitializeResponse(id int) InitializeResponse {
	return NewInitializeResponseWithID(mustMarshalID(id))
}

// NewInitializeResponseWithID creates an initialize response with a raw ID.
func NewInitializeResponseWithID(id json.RawMessage) InitializeResponse {
	return InitializeResponse{
		Response: NewResponse(id),
		Result: InitializeResult{
			Capabilities: ServerCapabilities{
				TextDocumentSync: TextDocumentSyncOptions{
					OpenClose: true,
					Change:    1,
				},
				HoverProvider:          true,
				DefinitionProvider:     true,
				ReferencesProvider:     true,
				DocumentSymbolProvider: true,
				FoldingRangeProvider:   true,
				CodeActionProvider:     true,
				CompletionProvider: map[string]any{
					"triggerCharacters": []string{"#", "[", "!", "`", ">", "-", "*", "_", "<"},
				},
			},
			ServerInfo: ServerInfo{
				Name:    "read_lsp",
				Version: "0.4.0",
			},
		},
	}
}

// mustMarshalID marshals an integer request ID or panics.
func mustMarshalID(id int) json.RawMessage {
	raw, err := json.Marshal(id)
	if err != nil {
		panic(err)
	}
	return raw
}
