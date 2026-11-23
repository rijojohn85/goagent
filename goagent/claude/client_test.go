package claude

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fakeAPI(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return NewClient("fake-api-key", server.URL, server.Client())
}

func TestSendReturnsReply(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "fake-api-key", r.Header.Get("x-api-key"))
		assert.Equal(t, "2026-10-21", r.Header.Get("anthropic-version"))
		assert.Equal(t, "application/json", r.Header.Get("content-type"))
		assert.Equal(t, "/v1/messages", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.JSONEq(t, `{"model":"claude-haiku-5-5","max_tokens":1024,"messages":[{"role":"user","content":[{"type":"text","text":"Hello"}]}]}`, string(body))
		assert.NoError(t, r.Body.Close())
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"response-id","model":"claude-haiku-5-5","content":[{"type":"text","text":"Hello, world!"}],"stop_reason":"stop","usage":{"input_tokens":5,"output_tokens":5}}`))
	}
	client := fakeAPI(t, handler)
	clientRequest := Request{Model: "claude-haiku-5-5", MaxTokens: 1024, Messages: []Message{{Role: RoleUser, Content: []ContentBlock{{Type: "text", Text: "Hello"}}}}}
	clientResponse, err := client.Send(t.Context(), clientRequest)
	require.NoError(t, err)
	require.Equal(t, "Hello, world!", clientResponse.Content[0].Text)
	require.Equal(t, Usage{InputTokens: 5, OutputTokens: 5}, clientResponse.Usage)
}

func TestSendReturnsAPIError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"type":"error","error":{"type":"invalid_request_error","message":"Invalid request"},"request_id":"req-123"}`))
	}
	client := fakeAPI(t, handler)
	clientRequest := Request{Model: "claude-haiku-5-5", MaxTokens: 1024, Messages: []Message{{Role: RoleUser, Content: []ContentBlock{{Type: "text", Text: "Hello"}}}}}
	_, err := client.Send(t.Context(), clientRequest)
	var apiErr *APIError

	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	assert.Equal(t, "invalid_request_error", apiErr.Type)
	assert.Equal(t, "Invalid request", apiErr.Message)
	assert.Equal(t, "req-123", apiErr.RequestID)
}
