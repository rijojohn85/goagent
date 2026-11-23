/* Package claude defines types and structures for interacting with the Claude API,
* including request and response formats, message roles, and content blocks.
 */
package claude

import (
	"strings"
)

const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}
type Response struct {
	ID         string         `json:"id"`
	Model      string         `json:"model"`
	Content    []ContentBlock `json:"content"`
	StopReason string         `json:"stop_reason"`
	Usage      Usage          `json:"usage"`
}
type Message struct {
	Role    string         `json:"role"`
	Content []ContentBlock `json:"content"`
}

type Request struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []Message `json:"messages"`
}

/* TextBlock creates a new ContentBlock of type "text" with the given text.
* Input: text - the text content for the block.
* Output: a ContentBlock with Type set to "text" and Text set to the provided text.
 */
func TextBlock(text string) ContentBlock {
	return ContentBlock{
		Type: "text",
		Text: text,
	}
}

/*Text returns the text content of the response,
* concatenating all text blocks into a single string.
* If there are no text blocks, it returns an empty string.
 */
func (r *Response) Text() string {
	var output strings.Builder
	for _, contentBlock := range r.Content {
		if contentBlock.Type == "text" {
			output.WriteString(contentBlock.Text)
			output.WriteString("\n")
		}
	}
	return output.String()
}
