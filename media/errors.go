package media

import (
	"fmt"
)

// MediaError represents an error from CDN operations.
type MediaError struct {
	StatusCode int
	Message    string
}

func (e *MediaError) Error() string {
	return fmt.Sprintf("media error: status=%d, message=%s", e.StatusCode, e.Message)
}

// IsClientError returns true if the CDN error is a client error (4xx).
func (e *MediaError) IsClientError() bool {
	return e.StatusCode >= 400 && e.StatusCode < 500
}

// IsServerError returns true if the CDN error is a server error (5xx).
func (e *MediaError) IsServerError() bool {
	return e.StatusCode >= 500
}

// CDNError is an alias for MediaError for backward compatibility.
type CDNError = MediaError
