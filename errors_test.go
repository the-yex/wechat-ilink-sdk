package ilinksdk

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/the-yex/wechat-ilink-sdk/ilink"
	"github.com/the-yex/wechat-ilink-sdk/login"
	"github.com/the-yex/wechat-ilink-sdk/media"
)

func TestWrapError_ClassifiesAuthenticationFailures(t *testing.T) {
	err := wrapError("get updates", &ilink.APIError{Code: 401, Message: "unauthorized"}, nil)

	require.Error(t, err)
	assert.True(t, IsAuthenticationError(err))
	assert.ErrorIs(t, err, ErrAuthenticationFailed)

	code, ok := ErrorCode(err)
	require.True(t, ok)
	assert.Equal(t, 401, code)
}

func TestWrapError_ClassifiesSessionExpired(t *testing.T) {
	err := wrapError("get updates", &ilink.APIError{Code: ilink.SessionExpiredErrCode, Message: "expired"}, nil)

	require.Error(t, err)
	assert.True(t, IsAuthenticationError(err))
	assert.ErrorIs(t, err, ErrSessionExpired)
}

func TestWrapError_ClassifiesMediaErrors(t *testing.T) {
	err := wrapError("upload media", &media.MediaError{StatusCode: 503, Message: "unavailable"}, ErrUploadFailed)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUploadFailed)
	assert.True(t, IsTemporaryError(err))

	code, ok := ErrorCode(err)
	require.True(t, ok)
	assert.Equal(t, 503, code)
}

func TestWrapError_PreservesContextCancellation(t *testing.T) {
	err := wrapError("send message", context.Canceled, nil)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)

	var sdkErr *Error
	assert.False(t, errors.As(err, &sdkErr))
}

func TestWrapError_ClassifiesLoginErrors(t *testing.T) {
	err := wrapError("login", login.ErrLoginCanceled, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrLoginCanceled)

	err = wrapError("login", login.ErrQRCodeExpired, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrQRCodeExpired)
}
