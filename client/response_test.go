package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrapResponseSuccess(t *testing.T) {
	data := "ok"

	resp := WrapResponse(0, "", 0, &data)

	assert.True(t, resp.Success)
	assert.Same(t, &data, resp.Data)
	assert.Nil(t, resp.Error)
}

func TestWrapResponseError(t *testing.T) {
	data := "ignored"

	resp := WrapResponse(1, "失败", 1001, &data)

	assert.False(t, resp.Success)
	assert.Nil(t, resp.Data)
	assert.Equal(t, &APIError{Code: 1, Message: "失败", ErrCode: 1001}, resp.Error)
}
