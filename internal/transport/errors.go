package transport

import (
	"fmt"
	"io"
	"strings"
	"unicode"
)

// MaxErrorMessageLength bounds the retained human-readable server message.
const MaxErrorMessageLength = 1024

// HTTPStatusError describes a server rejection of a heartbeat request.
type HTTPStatusError struct {
	StatusCode int
	RequestID  string
	Message    string
	Truncated  bool
}

// Error returns a safe summary of the HTTP rejection.
func (e *HTTPStatusError) Error() string {
	if e == nil {
		return "heartbeat request rejected"
	}
	message := fmt.Sprintf("heartbeat request rejected with HTTP %d", e.StatusCode)
	if e.Message != "" {
		message += ": " + e.Message
	}
	return message
}

func readErrorMessage(body io.Reader) (string, bool) {
	data, err := io.ReadAll(io.LimitReader(body, MaxResponseBodyBytes+1))
	truncated := len(data) > MaxResponseBodyBytes
	if truncated {
		data = data[:MaxResponseBodyBytes]
	}
	if err != nil {
		return "", truncated
	}

	safe := strings.Map(func(character rune) rune {
		if unicode.IsControl(character) {
			return ' '
		}
		return character
	}, string(data))
	safe = strings.Join(strings.Fields(safe), " ")
	runes := []rune(safe)
	if len(runes) > MaxErrorMessageLength {
		safe = string(runes[:MaxErrorMessageLength])
		truncated = true
	}
	return safe, truncated
}
