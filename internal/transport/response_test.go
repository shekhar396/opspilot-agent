package transport

import (
	"net/http"
	"strings"
	"testing"
)

func TestResponseAccepted(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent} {
		if !((Response{StatusCode: status}).Accepted()) {
			t.Errorf("status %d was not accepted", status)
		}
	}
	for _, status := range []int{http.StatusCreated, http.StatusBadRequest, http.StatusInternalServerError} {
		if (Response{StatusCode: status}).Accepted() {
			t.Errorf("status %d was unexpectedly accepted", status)
		}
	}
}

func TestHTTPStatusError(t *testing.T) {
	err := &HTTPStatusError{StatusCode: 401, RequestID: "request-1", Message: "unauthorized", Truncated: true}
	if !strings.Contains(err.Error(), "HTTP 401") || !strings.Contains(err.Error(), "unauthorized") {
		t.Fatalf("Error() = %q", err.Error())
	}
	if err.RequestID != "request-1" || !err.Truncated {
		t.Fatalf("error metadata = %#v", err)
	}
	if strings.Contains(err.Error(), "request-1") {
		t.Fatal("Error() unexpectedly exposes request ID")
	}

	empty := (&HTTPStatusError{StatusCode: 503}).Error()
	if empty != "heartbeat request rejected with HTTP 503" {
		t.Fatalf("empty-message Error() = %q", empty)
	}
}

func TestReadErrorMessageSanitizesAndBounds(t *testing.T) {
	message, truncated := readErrorMessage(strings.NewReader(" unsafe\n\tmessage\x00 "))
	if message != "unsafe message" || truncated {
		t.Fatalf("message = %q, truncated = %v", message, truncated)
	}

	message, truncated = readErrorMessage(strings.NewReader(strings.Repeat("x", MaxResponseBodyBytes+100)))
	if !truncated {
		t.Fatal("large message was not marked truncated")
	}
	if len([]rune(message)) > MaxErrorMessageLength {
		t.Fatalf("message length = %d, limit = %d", len([]rune(message)), MaxErrorMessageLength)
	}
}
