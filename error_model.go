package ilinksdk

import (
	"context"
	"errors"
	"fmt"

	"github.com/the-yex/wechat-ilink-sdk/ilink"
	"github.com/the-yex/wechat-ilink-sdk/internal/service"
	"github.com/the-yex/wechat-ilink-sdk/login"
	"github.com/the-yex/wechat-ilink-sdk/media"
)

// Error wraps SDK-facing failures with normalized metadata while preserving the
// original error for callers that need lower-level details.
type Error struct {
	Op        string
	Kind      error
	Code      int
	Temporary bool
	Err       error
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Op == "" {
		return e.Err.Error()
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

// Unwrap returns the underlying error.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// Is matches the normalized SDK kind first, then falls back to the wrapped error chain.
func (e *Error) Is(target error) bool {
	if e == nil {
		return target == nil
	}
	return (e.Kind != nil && target == e.Kind) || errors.Is(e.Err, target)
}

// IsAuthenticationError reports whether err represents an authentication or session-expiry failure.
func IsAuthenticationError(err error) bool {
	return errors.Is(err, ErrAuthenticationFailed) || errors.Is(err, ErrSessionExpired)
}

// IsTemporaryError reports whether err is classified as temporary/retryable.
func IsTemporaryError(err error) bool {
	var sdkErr *Error
	if errors.As(err, &sdkErr) {
		return sdkErr.Temporary
	}
	_, temporary, _ := classifyError(err, nil)
	return temporary
}

// ErrorCode returns the normalized API or CDN status code when available.
func ErrorCode(err error) (int, bool) {
	var sdkErr *Error
	if errors.As(err, &sdkErr) {
		if sdkErr.Code != 0 {
			return sdkErr.Code, true
		}
	}

	_, _, code := classifyError(err, nil)
	if code == 0 {
		return 0, false
	}
	return code, true
}

func wrapError(op string, err error, fallback error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}

	var sdkErr *Error
	if errors.As(err, &sdkErr) {
		if sdkErr.Op != "" || op == "" {
			return err
		}
		copied := *sdkErr
		copied.Op = op
		return &copied
	}

	kind, temporary, code := classifyError(err, fallback)
	return &Error{
		Op:        op,
		Kind:      kind,
		Code:      code,
		Temporary: temporary,
		Err:       err,
	}
}

func classifyError(err error, fallback error) (kind error, temporary bool, code int) {
	if err == nil {
		return nil, false, 0
	}

	var sdkErr *Error
	if errors.As(err, &sdkErr) {
		return sdkErr.Kind, sdkErr.Temporary, sdkErr.Code
	}

	switch {
	case errors.Is(err, ErrClientClosed):
		return ErrClientClosed, false, 0
	case errors.Is(err, ErrAuthenticationFailed):
		return ErrAuthenticationFailed, false, 0
	case errors.Is(err, ErrSessionExpired):
		return ErrSessionExpired, false, 0
	case errors.Is(err, service.ErrContextTokenRequired):
		return ErrContextTokenRequired, false, 0
	case errors.Is(err, login.ErrQRCodeExpired):
		return ErrQRCodeExpired, false, 0
	case errors.Is(err, login.ErrLoginCanceled):
		return ErrLoginCanceled, false, 0
	}

	var apiErr *ilink.APIError
	if errors.As(err, &apiErr) {
		code = apiErr.Code
		switch {
		case apiErr.IsSessionExpired():
			return ErrSessionExpired, false, code
		case apiErr.Code == 401:
			return ErrAuthenticationFailed, false, code
		case apiErr.Code >= 500:
			return fallback, true, code
		default:
			return fallback, false, code
		}
	}

	var mediaErr *media.MediaError
	if errors.As(err, &mediaErr) {
		return fallback, mediaErr.IsServerError(), mediaErr.StatusCode
	}

	return fallback, false, 0
}
