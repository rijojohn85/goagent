package claude

import (
	"encoding/json"
	"fmt"
)

type APIError struct {
	StatusCode int
	Type       string
	Message    string
	RequestID  string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d %s:%s RequestID: %s", e.StatusCode, e.Type, e.Message, e.RequestID)
}

func newAPIError(status int, body []byte) *APIError {
	var wire struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
		RequestID string `json:"request_id"`
	}

	if err := json.Unmarshal(body, &wire); err != nil || wire.Error.Type == "" {
		return &APIError{StatusCode: status, Message: string(body)}
	}
	return &APIError{
		StatusCode: status,
		Type:       wire.Error.Type,
		Message:    wire.Error.Message,
		RequestID:  wire.RequestID,
	}
}
