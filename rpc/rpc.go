package rpc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// BaseMessage is the common JSON-RPC envelope decoded before dispatch.
type BaseMessage struct {
	RPC    string          `json:"jsonrpc"`
	ID     json.RawMessage `json:"id,omitempty"`
	Method string          `json:"method"`
}

// IsRequest reports whether the message has a non-null request ID.
func (m BaseMessage) IsRequest() bool {
	return len(m.ID) > 0 && string(m.ID) != "null"
}

// RequestKey returns the request ID as a string key.
func (m BaseMessage) RequestKey() string {
	return string(m.ID)
}

// EncodeMessage serializes a message with an LSP Content-Length header.
func EncodeMessage(msg any) (string, error) {
	content, err := json.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("encode message: %w", err)
	}
	return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(content), content), nil
}

// DecodeMessage returns the method and body of an LSP message.
func DecodeMessage(msg []byte) (string, []byte, error) {
	baseMessage, content, err := DecodeMessageBase(msg)
	if err != nil {
		return "", nil, err
	}
	return baseMessage.Method, content, nil
}

// DecodeMessageBase decodes the JSON-RPC envelope and body.
func DecodeMessageBase(msg []byte) (BaseMessage, []byte, error) {
	header, content, found := bytes.Cut(msg, []byte{'\r', '\n', '\r', '\n'})
	if !found {
		return BaseMessage{}, nil, errors.New("DID NOT FIND SEAPRATOER")
	}

	contentLength, err := parseContentLength(header)
	if err != nil {
		return BaseMessage{}, nil, err
	}
	if len(content) < contentLength {
		return BaseMessage{}, nil, errors.New("message body shorter than content length")
	}

	var baseMessage BaseMessage
	if err := json.Unmarshal(content[:contentLength], &baseMessage); err != nil {
		return BaseMessage{}, nil, err
	}

	return baseMessage, content[:contentLength], nil
}

// Split is a bufio.Scanner split function for LSP frames.
func Split(data []byte, _ bool) (advance int, tocken []byte, err error) {
	header, content, found := bytes.Cut(data, []byte{'\r', '\n', '\r', '\n'})
	if !found {
		return 0, nil, nil
	}

	contentLength, err := parseContentLength(header)
	if err != nil {
		return 0, nil, err
	}

	if len(content) < contentLength {
		return 0, nil, nil
	}

	totalLength := len(header) + 4 + contentLength
	return totalLength, data[:totalLength], nil
}

// parseContentLength extracts the Content-Length header value.
func parseContentLength(header []byte) (int, error) {
	for _, line := range strings.Split(string(header), "\r\n") {
		name, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}

		if strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
			contentLength, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return 0, err
			}
			if contentLength < 0 {
				return 0, errors.New("content length cannot be negative")
			}
			return contentLength, nil
		}
	}

	return 0, errors.New("missing content length header")
}
