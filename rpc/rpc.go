package rpc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type BaseMessage struct {
	RPC    string          `json:"jsonrpc"`
	ID     json.RawMessage `json:"id,omitempty"`
	Method string          `json:"method"`
}

func (m BaseMessage) IsRequest() bool {
	return len(m.ID) > 0 && string(m.ID) != "null"
}

func (m BaseMessage) RequestKey() string {
	return string(m.ID)
}

func EncodeMessage(msg any) string {
	contnet, err := json.Marshal(msg)
	// TODO : handel this in better waay
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(contnet), contnet)
}

func DecodeMessage(msg []byte) (string, []byte, error) {
	baseMessage, content, err := DecodeMessageBase(msg)
	if err != nil {
		return "", nil, err
	}
	return baseMessage.Method, content, nil
}

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
