package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	DefaultBaseURL = "https://api.anthropic.com"
	apiVersion     = "2026-10-21"
)

type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func NewClient(apiKey, baseURL string, httpClient *http.Client) *Client {
	return &Client{
		apiKey:     apiKey,
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

func (c *Client) Send(ctx context.Context, req Request) (*Response, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	bodyBytes := bytes.NewReader(body)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/messages", bodyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	request.Header.Set("x-api-key", c.apiKey)
	request.Header.Set("anthropic-version", apiVersion)
	request.Header.Set("content-type", "application/json")

	resp, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, newAPIError(resp.StatusCode, raw)
	}
	var response Response
	err = json.Unmarshal(raw, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to Unmarshal response: %w", err)
	}
	return &response, nil
}
