package rpc_test

import (
	"bufio"
	"github.com/atharva-0123-cs/read_lsp/rpc"
	"strings"
	"testing"
)

type EncodingExample struct {
	Testing bool
}

func TestEncode(t *testing.T) {
	expected := "Content-Length: 16\r\n\r\n{\"Testing\":true}"
	actual, err := rpc.EncodeMessage(EncodingExample{Testing: true})
	if err != nil {
		t.Fatal(err)
	}

	if actual != expected {
		t.Fatalf("Expected : %s , Actual : %s ", expected, actual)
	}
}

func TestEncodeRejectsUnsupportedValue(t *testing.T) {
	encoded, err := rpc.EncodeMessage(make(chan int))
	if err == nil {
		t.Fatal("expected an encoding error")
	}
	if encoded != "" {
		t.Fatalf("expected no encoded message, got %q", encoded)
	}
}

func TestDecode(t *testing.T) {
	incomingMessage := "Content-Length: 15\r\n\r\n{\"method\":\"hi\"}"
	method, content, err := rpc.DecodeMessage([]byte(incomingMessage))
	contentLength := len(content)
	if err != nil {
		t.Fatal(err)
	}

	if contentLength != 15 {
		t.Fatalf("contentLength : 16, Got : %d ", contentLength)
	}

	if method != "hi" {
		t.Fatalf("Expected : 'hi', Got : %s ", method)
	}
}

func TestDecodeWithAdditionalHeaders(t *testing.T) {
	incomingMessage := "Content-Type: application/vscode-jsonrpc; charset=utf-8\r\nContent-Length: 31\r\n\r\n{\"jsonrpc\":\"2.0\",\"method\":\"hi\"}"
	method, content, err := rpc.DecodeMessage([]byte(incomingMessage))
	if err != nil {
		t.Fatal(err)
	}

	if method != "hi" {
		t.Fatalf("Expected : 'hi', Got : %s ", method)
	}

	if string(content) != "{\"jsonrpc\":\"2.0\",\"method\":\"hi\"}" {
		t.Fatalf("Unexpected content: %s", content)
	}
}

func TestDecodeRejectsShortBody(t *testing.T) {
	incomingMessage := "Content-Length: 100\r\n\r\n{\"method\":\"hi\"}"
	_, _, err := rpc.DecodeMessage([]byte(incomingMessage))
	if err == nil {
		t.Fatal("Expected an error for short body")
	}
}

func TestSplitWaitsForCompleteMessage(t *testing.T) {
	data := []byte("Content-Length: 15\r\n\r\n{\"method\":\"hi\"}")
	advance, token, err := rpc.Split(data[:len(data)-2], false)
	if err != nil {
		t.Fatal(err)
	}
	if advance != 0 || token != nil {
		t.Fatalf("Expected scanner to wait, advance=%d token=%q", advance, token)
	}

	advance, token, err = rpc.Split(data, false)
	if err != nil {
		t.Fatal(err)
	}
	if advance != len(data) {
		t.Fatalf("Expected advance %d, got %d", len(data), advance)
	}
	if string(token) != string(data) {
		t.Fatalf("Unexpected token: %s", token)
	}
}

func TestScannerWithSplit(t *testing.T) {
	first := "Content-Length: 15\r\n\r\n{\"method\":\"hi\"}"
	second := "Content-Length: 16\r\n\r\n{\"method\":\"bye\"}"
	scanner := bufio.NewScanner(stringsReader(first + second))
	scanner.Split(rpc.Split)

	count := 0
	for scanner.Scan() {
		count++
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("Expected 2 messages, got %d", count)
	}
}

func stringsReader(s string) *strings.Reader {
	return strings.NewReader(s)
}
