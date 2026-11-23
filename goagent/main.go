package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rijojohn85/goagent/claude"
)

const (
	model        = "claude-haiku-5-5"
	maxTokens    = 1024
	apiURL       = "https://api.anthropic.com"
	apiKeyEnvVar = "ANTHROPIC_API_KEY"
)

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
	client := claude.NewClient(apiKey, apiURL, &http.Client{Timeout: time.Minute * 2})
	err := run(context.Background(), client, prompt, os.Stdout, os.Stderr)
	if err != nil {
		fail("error: %v", err)
	}
}

type messageSender interface {
	Send(ctx context.Context, req claude.Request) (*claude.Response, error)
}

func run(ctx context.Context, sender messageSender, prompt string, stdout, stderr io.Writer) error {
	contentBlock := claude.TextBlock(prompt)
	message := claude.Message{
		Role:    claude.RoleUser,
		Content: []claude.ContentBlock{contentBlock},
	}
	req := claude.Request{
		Model:     model,
		MaxTokens: maxTokens,
		Messages:  []claude.Message{message},
	}
	response, err := sender.Send(ctx, req)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "%s", response.Text())
	fmt.Fprintf(stderr, "[stop: %s, tokens in: %d, out: %d]\n",
		response.StopReason, response.Usage.InputTokens, response.Usage.OutputTokens)
	return nil
}
