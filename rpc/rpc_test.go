package rpc_test

import (
	"golsp/rpc"
	"testing"
)

type EncodingExaple struct {
	Testing bool
}

func TestEncode(t *testing.T) {
	expected := "Content-Length: 16\r\n\r\n{\"Testing\":true}"
	actual := rpc.EncodeMessage(EncodingExaple{Testing: true})

	if actual != expected {
		t.Fatalf("Expected : %s , Actual : %s ", expected, actual)
	}
}

func TestDecode(t *testing.T) {
	incommingMeaasge := "Content-Length: 15\r\n\r\n{\"Method\":\"hi\"}"
	method, content, err := rpc.DecodeMessage([]byte(incommingMeaasge))
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
