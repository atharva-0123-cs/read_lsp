package lsp_test

import (
	"encoding/json"
	"github.com/atharva-0123-cs/read_lsp/lsp"
	"testing"
)

func TestNewInitializeResponseWithID(t *testing.T) {
	response := lsp.NewInitializeResponseWithID(json.RawMessage(`"abc-123"`))

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}

	var decoded struct {
		RPC    string `json:"jsonrpc"`
		ID     string `json:"id"`
		Result struct {
			ServerInfo struct {
				Name string `json:"name"`
			} `json:"serverInfo"`
			Capabilities struct {
				HoverProvider          bool           `json:"hoverProvider"`
				DefinitionProvider     bool           `json:"definitionProvider"`
				ReferencesProvider     bool           `json:"referencesProvider"`
				DocumentSymbolProvider bool           `json:"documentSymbolProvider"`
				FoldingRangeProvider   bool           `json:"foldingRangeProvider"`
				CodeActionProvider     bool           `json:"codeActionProvider"`
				CompletionProvider     map[string]any `json:"completionProvider"`
			} `json:"capabilities"`
		} `json:"result"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.RPC != "2.0" {
		t.Fatalf("Expected jsonrpc 2.0, got %q", decoded.RPC)
	}
	if decoded.ID != "abc-123" {
		t.Fatalf("Expected request id to be preserved, got %q", decoded.ID)
	}
	if decoded.Result.ServerInfo.Name != "read_lsp" {
		t.Fatalf("Expected server name read_lsp, got %q", decoded.Result.ServerInfo.Name)
	}
	if !decoded.Result.Capabilities.HoverProvider ||
		!decoded.Result.Capabilities.DefinitionProvider ||
		!decoded.Result.Capabilities.ReferencesProvider ||
		!decoded.Result.Capabilities.DocumentSymbolProvider ||
		!decoded.Result.Capabilities.FoldingRangeProvider ||
		!decoded.Result.Capabilities.CodeActionProvider {
		t.Fatal("Expected core LSP providers to be enabled")
	}
	if decoded.Result.Capabilities.CompletionProvider == nil {
		t.Fatal("Expected completion provider options")
	}
}
