package transport

import "net/http"

// Response contains bounded metadata from a heartbeat response.
type Response struct {
	StatusCode int
	RequestID  string
}

// Accepted reports whether the response uses an explicitly supported status.
func (r Response) Accepted() bool {
	switch r.StatusCode {
	case http.StatusOK, http.StatusAccepted, http.StatusNoContent:
		return true
	default:
		return false
	}
}
