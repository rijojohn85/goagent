package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	model           = "claude-haiku-5-5"
	maxTokens       = 1024
	apiURL          = "https://api.anthropic.com/v1/messages"
	apiVersionKey   = "anthropic-version"
	apiVersionValue = "2026-10-21"
	apiKeyKey       = "x-api-key"
	apiKeyEnvVar    = "ANTHROPIC_API_KEY"
)

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
type usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}
type messageResponse struct {
	Content    []contentBlock `json:"content"`
	StopReason string         `json:"stop_reason"`
	Usage      usage          `json:"usage"`
}
type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type messageRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []message `json:"messages"`
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "goagent: "+format+"\n", args...)
	os.Exit(1)
}

func main() {
	apiKey := os.Getenv(apiKeyEnvVar)
	if len(apiKey) == 0 {
		fail("Anthropic API key not set, export it as %s", apiKeyEnvVar)
	}
	words := os.Args[1:]
	if len(words) == 0 {
		fail("useage: goagent 'your question'")
	}
	prompt := strings.Join(words, " ")

	m := message{Role: "user", Content: prompt}
	mr := messageRequest{
		Model:     model,
		MaxTokens: maxTokens,
		Messages:  []message{m},
	}
	body, err := json.Marshal(mr)
	if err != nil {
		fail("encoding request: %v", err)
	}
	bodyBytes := bytes.NewReader(body)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, apiURL, bodyBytes)
	if err != nil {
		fail("error creating request: %v", err)
	}
	req.Header.Set(apiKeyKey, apiKey)
	req.Header.Set(apiVersionKey, apiVersionValue)
	req.Header.Set("content-type", "application/json")

	client := http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		fail("error sending api request: %v", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		fail("error reading api response: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		fail("API returned %s: %s", resp.StatusCode, raw)
	}
	var response messageResponse
	err = json.Unmarshal(raw, &response)
	if err != nil {
		fail("Error Unmarshalling json response: %v", err)
	}

	for _, block := range response.Content {
		if block.Type == "text" {
			fmt.Println(block.Text)
		}
	}
	fmt.Fprintf(os.Stderr, "[stop: %s, tokens in: %d, out: %d]\n",
		response.StopReason, response.Usage.InputTokens, response.Usage.OutputTokens)
}
