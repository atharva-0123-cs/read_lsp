package server_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"strconv"
	"strings"
	"testing"

	"github.com/atharva-0123-cs/read_lsp/lsp"
	"github.com/atharva-0123-cs/read_lsp/rpc"
	"github.com/atharva-0123-cs/read_lsp/server"
)

type wireMessage struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code int `json:"code"`
	} `json:"error"`
	Params struct {
		URI         string           `json:"uri"`
		Diagnostics []lsp.Diagnostic `json:"diagnostics"`
	} `json:"params"`
}

func TestServerHandlesConcurrentRequestsAndDiagnostics(t *testing.T) {
	uri := "file:///example.md"
	request := []any{
		initializeRequest(1),
		notification("initialized", map[string]any{}),
		notification("textDocument/didOpen", map[string]any{
			"textDocument": map[string]any{
				"uri":        uri,
				"languageId": "markdown",
				"version":    1,
				"text":       "#BadHeading\n",
			},
		}),
	}

	for id := 10; id < 22; id++ {
		request = append(request, hoverRequest(id, uri))
	}

	request = append(request,
		completionRequest(30, uri),
		shutdownRequest(2),
		notification("exit", map[string]any{}),
	)

	messages := runServer(t, request)
	responseIDs := map[string]bool{}
	for _, message := range messages {
		if len(message.ID) > 0 {
			responseIDs[string(message.ID)] = true
		}
	}

	for _, id := range []string{"1", "2", "30"} {
		if !responseIDs[id] {
			t.Fatalf("expected response for id %s, got %#v", id, responseIDs)
		}
	}
	for id := 10; id < 22; id++ {
		key := strconv.Itoa(id)
		if !responseIDs[key] {
			t.Fatalf("expected response for id %s, got %#v", key, responseIDs)
		}
	}

	if string(messages[0].ID) != "1" {
		t.Fatalf("expected initialize response first, got %#v", messages[0])
	}

	if !hasNonEmptyDiagnostics(messages, uri) {
		t.Fatal("expected diagnostics for malformed Markdown")
	}
}

func TestServerPublishesUpdatedDiagnosticsAfterChange(t *testing.T) {
	uri := "file:///changed.md"
	request := []any{
		initializeRequest(1),
		notification("textDocument/didOpen", map[string]any{
			"textDocument": map[string]any{
				"uri":        uri,
				"languageId": "markdown",
				"version":    1,
				"text":       "# Example\n",
			},
		}),
		notification("textDocument/didChange", map[string]any{
			"textDocument": map[string]any{"uri": uri, "version": 2},
			"contentChanges": []any{
				map[string]any{"text": "#BadHeading\n"},
			},
		}),
		shutdownRequest(2),
		notification("exit", map[string]any{}),
	}

	messages := runServer(t, request)
	for _, message := range messages {
		if message.Method != "textDocument/publishDiagnostics" || message.Params.URI != uri {
			continue
		}
		for _, diagnostic := range message.Params.Diagnostics {
			if strings.Contains(diagnostic.Message, "Add a space") {
				return
			}
		}
	}

	t.Fatal("expected a diagnostic after the document changed")
}

func TestServerClearsDiagnosticsWhenDocumentCloses(t *testing.T) {
	uri := "file:///closed.md"
	request := []any{
		initializeRequest(1),
		notification("textDocument/didOpen", map[string]any{
			"textDocument": map[string]any{
				"uri":        uri,
				"languageId": "markdown",
				"version":    1,
				"text":       "#BadHeading\n",
			},
		}),
		notification("textDocument/didClose", map[string]any{
			"textDocument": map[string]any{"uri": uri},
		}),
		shutdownRequest(2),
		notification("exit", map[string]any{}),
	}

	messages := runServer(t, request)

	var last []lsp.Diagnostic
	found := false
	for _, message := range messages {
		if message.Method == "textDocument/publishDiagnostics" && message.Params.URI == uri {
			last = message.Params.Diagnostics
			found = true
		}
	}

	if !found {
		t.Fatal("expected publishDiagnostics notification")
	}
	if len(last) != 0 {
		t.Fatalf("expected diagnostics to be cleared, got %#v", last)
	}
}

func TestServerReturnsMarkdownCompletionAndCodeAction(t *testing.T) {
	uri := "file:///features.md"
	request := []any{
		initializeRequest(1),
		notification("textDocument/didOpen", map[string]any{
			"textDocument": map[string]any{
				"uri":        uri,
				"languageId": "markdown",
				"version":    1,
				"text":       "#BadHeading\n",
			},
		}),
		completionRequestAt(30, uri, 0, 1),
		codeActionRequest(31, uri, 0, 0, 0, 4),
		shutdownRequest(2),
		notification("exit", map[string]any{}),
	}

	messages := runServer(t, request)

	var completions []lsp.CompletionItem
	var actions []lsp.CodeAction
	for _, message := range messages {
		switch string(message.ID) {
		case "30":
			if err := json.Unmarshal(message.Result, &completions); err != nil {
				t.Fatal(err)
			}
		case "31":
			if err := json.Unmarshal(message.Result, &actions); err != nil {
				t.Fatal(err)
			}
		}
	}

	if len(completions) != 6 {
		t.Fatalf("expected heading completions, got %#v", completions)
	}
	if completions[0].TextEdit == nil || completions[0].InsertTextFormat != 2 {
		t.Fatalf("expected snippet completion, got %#v", completions[0])
	}
	if len(actions) == 0 {
		t.Fatal("expected a heading code action")
	}
}

func TestServerReturnsNavigationFeatures(t *testing.T) {
	uri := "file:///navigation.md"
	request := []any{
		initializeRequest(1),
		notification("textDocument/didOpen", map[string]any{
			"textDocument": map[string]any{
				"uri":        uri,
				"languageId": "markdown",
				"version":    1,
				"text":       "# Target Heading\n\nSee [jump](#target-heading).\n",
			},
		}),
		documentSymbolRequest(30, uri),
		foldingRangeRequest(31, uri),
		referencesRequest(32, uri, 2, 6, true),
		shutdownRequest(2),
		notification("exit", map[string]any{}),
	}

	messages := runServer(t, request)

	var symbols []lsp.DocumentSymbol
	var folds []lsp.FoldingRange
	var references []lsp.Location
	for _, message := range messages {
		switch string(message.ID) {
		case "30":
			if err := json.Unmarshal(message.Result, &symbols); err != nil {
				t.Fatal(err)
			}
		case "31":
			if err := json.Unmarshal(message.Result, &folds); err != nil {
				t.Fatal(err)
			}
		case "32":
			if err := json.Unmarshal(message.Result, &references); err != nil {
				t.Fatal(err)
			}
		}
	}

	if len(symbols) != 1 || symbols[0].Name != "Target Heading" {
		t.Fatalf("unexpected document symbols: %#v", symbols)
	}
	if len(folds) == 0 {
		t.Fatal("expected folding ranges")
	}
	if len(references) != 2 {
		t.Fatalf("expected heading and anchor reference, got %#v", references)
	}
}

func TestServerCancelsRequestedWork(t *testing.T) {
	uri := "file:///cancel.md"
	request := []any{
		initializeRequest(1),
		notification("textDocument/didOpen", map[string]any{
			"textDocument": map[string]any{
				"uri":        uri,
				"languageId": "markdown",
				"version":    1,
				"text":       "# Heading\n",
			},
		}),
		notification("$/cancelRequest", map[string]any{"id": 50}),
		hoverRequest(50, uri),
		shutdownRequest(2),
		notification("exit", map[string]any{}),
	}

	messages := runServer(t, request)
	for _, message := range messages {
		if string(message.ID) != "50" {
			continue
		}
		if message.Error == nil || message.Error.Code != -32800 {
			t.Fatalf("expected cancellation error, got %#v", message)
		}
		return
	}

	t.Fatal("expected a response for the cancelled request")
}

func TestServerReturnsMethodNotFoundForUnknownRequest(t *testing.T) {
	request := []any{
		initializeRequest(1),
		map[string]any{
			"jsonrpc": "2.0",
			"id":      7,
			"method":  "workspace/unknown",
			"params":  map[string]any{},
		},
		shutdownRequest(2),
		notification("exit", map[string]any{}),
	}

	messages := runServer(t, request)
	for _, message := range messages {
		if string(message.ID) == "7" {
			if message.Error == nil || message.Error.Code != -32601 {
				t.Fatalf("expected method not found error, got %#v", message)
			}
			return
		}
	}

	t.Fatal("expected a response for the unknown request")
}

func runServer(t *testing.T, messages []any) []wireMessage {
	t.Helper()

	var input strings.Builder
	for _, message := range messages {
		input.WriteString(rpc.EncodeMessage(message))
	}

	var output bytes.Buffer
	srv := server.New(server.Options{
		Logger:              log.New(io.Discard, "", 0),
		Workers:             4,
		DiagnosticsDebounce: 0,
	})

	if err := srv.Serve(context.Background(), strings.NewReader(input.String()), &output); err != nil {
		t.Fatal(err)
	}

	return decodeOutput(t, output.String())
}

func decodeOutput(t *testing.T, output string) []wireMessage {
	t.Helper()

	scanner := bufio.NewScanner(strings.NewReader(output))
	scanner.Split(rpc.Split)

	messages := []wireMessage{}
	for scanner.Scan() {
		_, content, err := rpc.DecodeMessage(scanner.Bytes())
		if err != nil {
			t.Fatal(err)
		}

		var message wireMessage
		if err := json.Unmarshal(content, &message); err != nil {
			t.Fatal(err)
		}
		messages = append(messages, message)
	}

	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}

	return messages
}

func hasNonEmptyDiagnostics(messages []wireMessage, uri string) bool {
	for _, message := range messages {
		if message.Method == "textDocument/publishDiagnostics" &&
			message.Params.URI == uri &&
			len(message.Params.Diagnostics) > 0 {
			return true
		}
	}
	return false
}

func initializeRequest(id int) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "initialize",
		"params": map[string]any{
			"clientInfo": map[string]any{"name": "test", "version": "1.0"},
		},
	}
}

func shutdownRequest(id int) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "shutdown",
		"params":  nil,
	}
}

func hoverRequest(id int, uri string) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "textDocument/hover",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 0, "character": 0},
		},
	}
}

func completionRequest(id int, uri string) map[string]any {
	return completionRequestAt(id, uri, 0, 0)
}

func completionRequestAt(id int, uri string, line, character int) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "textDocument/completion",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": line, "character": character},
		},
	}
}

func codeActionRequest(id int, uri string, startLine, startCharacter, endLine, endCharacter int) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "textDocument/codeAction",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"range": map[string]any{
				"start": map[string]any{"line": startLine, "character": startCharacter},
				"end":   map[string]any{"line": endLine, "character": endCharacter},
			},
			"context": map[string]any{},
		},
	}
}

func documentSymbolRequest(id int, uri string) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "textDocument/documentSymbol",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
		},
	}
}

func foldingRangeRequest(id int, uri string) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "textDocument/foldingRange",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
		},
	}
}

func referencesRequest(id int, uri string, line, character int, includeDeclaration bool) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "textDocument/references",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": line, "character": character},
			"context":      map[string]any{"includeDeclaration": includeDeclaration},
		},
	}
}

func notification(method string, params any) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	}
}
