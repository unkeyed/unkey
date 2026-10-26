package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Payload represents a Slack webhook message payload.
type Payload struct {
	Text   string  `json:"text"`
	Blocks []Block `json:"blocks"`
}

// Block represents a Slack block element.
type Block struct {
	Type     string  `json:"type"`
	Text     *Text   `json:"text,omitempty"`
	Fields   []Field `json:"fields,omitempty"`
	Elements []Text  `json:"elements,omitempty"`
}

// Text represents a Slack text element.
type Text struct {
	Type     string `json:"type"`
	Text     string `json:"text"`
	Emoji    bool   `json:"emoji,omitempty"`
	Verbatim bool   `json:"verbatim,omitempty"`
}

// Field represents a Slack section field.
type Field struct {
	Type     string `json:"type"`
	Text     string `json:"text"`
	Verbatim bool   `json:"verbatim,omitempty"`
}

// NewHeaderBlock creates a header block with plain text.
func NewHeaderBlock(text string) Block {
	return Block{
		Type: "header",
		Text: &Text{
			Type:     "plain_text",
			Text:     text,
			Emoji:    true,
			Verbatim: false,
		},
		Fields:   nil,
		Elements: nil,
	}
}

// NewSectionBlock creates a section block with markdown fields.
func NewSectionBlock(fields ...Field) Block {
	return Block{
		Type:     "section",
		Text:     nil,
		Fields:   fields,
		Elements: nil,
	}
}

// NewMarkdownField creates a markdown field for use in section blocks.
func NewMarkdownField(text string) Field {
	return Field{
		Type:     "mrkdwn",
		Text:     text,
		Verbatim: false,
	}
}

func NewMarkdownBlock(text string) Block {
	return Block{
		Type:     "section",
		Text:     &Text{Type: "mrkdwn", Text: text, Emoji: false, Verbatim: true},
		Fields:   nil,
		Elements: nil,
	}
}

func NewContextBlock(text string) Block {
	return Block{
		Type:     "context",
		Text:     nil,
		Fields:   nil,
		Elements: []Text{{Type: "plain_text", Text: text, Emoji: false, Verbatim: false}},
	}
}

func NewMarkdownContextBlock(text string) Block {
	return Block{
		Type:     "context",
		Text:     nil,
		Fields:   nil,
		Elements: []Text{{Type: "mrkdwn", Text: text, Emoji: false, Verbatim: true}},
	}
}

// Client sends messages to Slack webhooks.
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new Slack webhook client.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// Send posts a payload to the given webhook URL.
func (c *Client) Send(ctx context.Context, webhookURL string, payload Payload) (err error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close response body: %w", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack returned status %d", resp.StatusCode)
	}

	return nil
}
