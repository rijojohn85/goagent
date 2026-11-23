package main

import (
	"bytes"
	"context"
	"testing"

	"github.com/rijojohn85/goagent/claude"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSender struct {
	got      claude.Request
	response *claude.Response
	err      error
}

func (f *fakeSender) Send(ctx context.Context, req claude.Request) (*claude.Response, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	f.got = req
	return f.response, f.err
}

func TestRunPrintsAnswer(t *testing.T) {
	response := &claude.Response{
		ID:         "response-id",
		Model:      model,
		Content:    []claude.ContentBlock{{Type: "text", Text: "The capital of India is New Delhi"}},
		StopReason: "stop",
		Usage:      claude.Usage{InputTokens: 5, OutputTokens: 5},
	}
	sender := fakeSender{
		response: response,
		err:      nil,
	}

	outBuf := bytes.Buffer{}
	errBuf := bytes.Buffer{}
	err := run(t.Context(), &sender, "What is the capital of India?", &outBuf, &errBuf)
	require.NoError(t, err)
	assert.Equal(t, "The capital of India is New Delhi\n", outBuf.String())
	assert.Equal(t, "[stop: stop, tokens in: 5, out: 5]\n", errBuf.String())
}

func TestRunReturnsError(t *testing.T) {
	sender := fakeSender{
		err: &claude.APIError{StatusCode: 404, Type: "not_found", Message: "Resource not found", RequestID: "req-123"},
	}
	outBuf := bytes.Buffer{}
	errBuf := bytes.Buffer{}
	err := run(t.Context(), &sender, "What is the capital of India?", &outBuf, &errBuf)
	var apiErr *claude.APIError
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, 404, err.(*claude.APIError).StatusCode)
	require.Equal(t, "not_found", err.(*claude.APIError).Type)
	require.Equal(t, "Resource not found", err.(*claude.APIError).Message)
	require.Equal(t, "req-123", err.(*claude.APIError).RequestID)
}
