package common

import (
	"fmt"
	"strings"
)

const (
	MessageFormat    = "%s||%s||%s" // message key || message type || message body
	MessageSeparator = "||"
)

type MessageType string

const (
	Connect MessageType = "0"
	Body    MessageType = "1"
	Close   MessageType = "2"
)

func FormatMessage(msgKey string, messageType MessageType, body string) string {
	return fmt.Sprintf(MessageFormat, msgKey, messageType, body)
}

func ParseMessage(rawMessage string) Message {
	parts := strings.SplitN(string(rawMessage), MessageSeparator, 3)
	return Message{
		Key:  parts[0],
		Type: MessageType(parts[1]),
		Body: parts[2],
	}
}

type Message struct {
	Key  string
	Type MessageType
	Body string
}

type HTTPMessage struct {
	Hijack  bool                `json:"hijack,omitempty"`
	Host    string              `json:"host,omitempty"`
	Method  string              `json:"method,omitempty"`
	URL     string              `json:"url,omitempty"`
	Headers map[string][]string `json:"headers,omitempty"`
	Code    int                 `json:"code,omitempty"`
	Body    []byte              `json:"body,omitempty"`
	EOF     bool                `json:"eof,omitempty"`
}
