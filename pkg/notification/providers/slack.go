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

const slackContentMax = 40000

type slack struct {
	client *http.Client
}

type slackPayload struct {
	Text string `json:"text"`
}

var slackPath = regexp.MustCompile(`^/services/[A-Za-z0-9]+/[A-Za-z0-9]+/[A-Za-z0-9]+$`)

// NewSlack sends to Slack incoming webhooks supplied in Target.URL.
// Slack provides no idempotency window for these sends.
func NewSlack() notification.Provider { return &slack{client: newHTTPClient()} }

func (p *slack) Name() string { return "slack" }

func (p *slack) Channel() notification.Channel { return notification.Slack }

func (p *slack) IdempotencyWindow() time.Duration { return 0 }

func (p *slack) Send(ctx context.Context, request notification.Request) (notification.Receipt, error) {
	endpoint, err := validateURL(request.Target.URL)
	if err != nil || endpoint.Host != "hooks.slack.com" || !slackPath.MatchString(endpoint.EscapedPath()) || endpoint.RawQuery != "" {
		return notification.Receipt{}, sendError(notification.Rejected, "invalid_webhook_url")
	}
	text := request.Content.Text
	if request.Content.Subject != "" {
		text = request.Content.Subject + "\n" + text
	}
	if utf8.RuneCountInString(text) > slackContentMax {
		return notification.Receipt{}, sendError(notification.Rejected, "content_too_large")
	}
	body, err := json.Marshal(slackPayload{Text: escapeSlack(text)})
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
	if strings.TrimSpace(string(responseBody)) != "ok" {
		return notification.Receipt{}, sendError(notification.Ambiguous, "invalid_response")
	}
	return notification.Receipt{MessageID: ""}, nil
}

func escapeSlack(value string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return replacer.Replace(value)
}
