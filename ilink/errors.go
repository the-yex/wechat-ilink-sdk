package ilink

import (
	"fmt"
)

// APIError represents an error returned by the WeChat API.
type APIError struct {
	Code    int
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("api error: code=%d, message=%s", e.Code, e.Message)
}

// IsSessionExpired returns true if the error indicates session expiry.
func (e *APIError) IsSessionExpired() bool {
	return e.Code == SessionExpiredErrCode
}