package lsp

import "encoding/json"

type InitializeRequest struct {
	Request
	Params InitializeRequestParams `json:"params"`
}

type InitializeRequestParams struct {
	ClientInfo *ClientInfo `json:"clientInfo"`

	// ... there's tons more goes here
}

type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type InitializeResponse struct {
	Response
	Result InitializeResult `json:"result"`
}

type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   ServerInfo         `json:"serverInfo"`
}

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

type TextDocumentSyncOptions struct {
	OpenClose bool `json:"openClose"`
	Change    int  `json:"change"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func NewInitializeResponse(id int) InitializeResponse {
	return NewInitializeResponseWithID(mustMarshalID(id))
}

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

func mustMarshalID(id int) json.RawMessage {
	raw, err := json.Marshal(id)
	if err != nil {
		panic(err)
	}
	return raw
}
