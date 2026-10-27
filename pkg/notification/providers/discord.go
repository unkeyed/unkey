package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/unkeyed/unkey/pkg/notification"
)

var discordPath = regexp.MustCompile(`^/api(?:/v[0-9]+)?/webhooks/[0-9]+/[A-Za-z0-9._-]+$`)

const discordContentMax = 2000

type discord struct {
	client *http.Client
}

type discordPayload struct {
	Content         string                 `json:"content"`
	AllowedMentions discordAllowedMentions `json:"allowed_mentions"`
}

type discordAllowedMentions struct {
	Parse []string `json:"parse"`
}

type discordResponse struct {
	ID string `json:"id"`
}

// NewDiscord sends to Discord webhooks supplied in Target.URL with mentions disabled.
// Discord provides no idempotency window for these sends.
func NewDiscord() notification.Provider { return &discord{client: newHTTPClient()} }

func (p *discord) Name() string { return "discord" }

func (p *discord) Channel() notification.Channel { return notification.Discord }

func (p *discord) IdempotencyWindow() time.Duration { return 0 }

func (p *discord) Send(ctx context.Context, request notification.Request) (notification.Receipt, error) {
	endpoint, err := validateURL(request.Target.URL)
	validHost := endpoint != nil && (endpoint.Host == "discord.com" || endpoint.Host == "discordapp.com")
	if err != nil || !validHost || !discordPath.MatchString(endpoint.EscapedPath()) || endpoint.RawQuery != "" {
		return notification.Receipt{}, sendError(notification.Rejected, "invalid_webhook_url")
	}
	endpoint.RawQuery = "wait=true"
	content := request.Content.Text
	if request.Content.Subject != "" {
		content = request.Content.Subject + "\n" + content
	}
	if utf8.RuneCountInString(content) > discordContentMax {
		return notification.Receipt{}, sendError(notification.Rejected, "content_too_large")
	}
	payload := discordPayload{
		Content:         content,
		AllowedMentions: discordAllowedMentions{Parse: []string{}},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return notification.Receipt{}, sendError(notification.Rejected, "invalid_content")
	}
	httpRequest, err := newJSONRequest(ctx, endpoint.String(), body)
	if err != nil {
		return notification.Receipt{}, sendError(notification.Rejected, "invalid_request")
	}
	responseBody, status, headers, err := send(p.client, httpRequest)
	if err != nil {
		return notification.Receipt{}, err
	}
	if status < 200 || status >= 300 {
		return notification.Receipt{}, statusError(status, headers)
	}
	var response discordResponse
	if json.Unmarshal(responseBody, &response) != nil || strings.TrimSpace(response.ID) == "" {
		return notification.Receipt{}, sendError(notification.Ambiguous, "invalid_response")
	}
	return notification.Receipt{MessageID: response.ID}, nil
}
