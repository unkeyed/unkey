package providers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/notification"
)

const resendEndpoint = "https://api.resend.com/emails"

var deliveryIDPattern = regexp.MustCompile(`^ndlv_[0-9a-f]{40}$`)

type resend struct {
	apiKey string
	from   string
	client *http.Client
}

type resendPayload struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Text    string `json:"text"`
	HTML    string `json:"html,omitempty"`
}

type resendResponse struct {
	ID string `json:"id"`
}

type resendErrorResponse struct {
	Name string `json:"name"`
}

// NewResend requires an API key and sender address, returning nil on invalid configuration.
// DeliveryID is used as the idempotency key with a 24-hour retention window.
func NewResend(apiKey, from string) (notification.Provider, error) {
	if apiKey == "" || strings.TrimSpace(apiKey) != apiKey || strings.ContainsAny(apiKey, "\r\n") {
		return nil, errors.New("invalid Resend API key")
	}
	if _, err := mail.ParseAddress(from); err != nil || strings.ContainsAny(from, "\r\n") {
		return nil, errors.New("invalid Resend sender")
	}
	return &resend{apiKey: apiKey, from: from, client: newHTTPClient()}, nil
}

func (p *resend) Name() string { return "resend" }

func (p *resend) Channel() notification.Channel { return notification.Email }

func (p *resend) IdempotencyWindow() time.Duration { return 24 * time.Hour }

func (p *resend) Send(ctx context.Context, request notification.Request) (notification.Receipt, error) {
	if !deliveryIDPattern.MatchString(request.DeliveryID) {
		return notification.Receipt{}, sendError(notification.Rejected, "invalid_delivery_id")
	}
	address, err := mail.ParseAddress(request.Target.Email)
	if err != nil || address.Address != request.Target.Email || strings.ContainsAny(request.Target.Email, "\r\n") {
		return notification.Receipt{}, sendError(notification.Rejected, "invalid_email")
	}
	body, err := json.Marshal(resendPayload{
		From:    p.from,
		To:      request.Target.Email,
		Subject: request.Content.Subject,
		Text:    request.Content.Text,
		HTML:    request.Content.HTML,
	})
	if err != nil {
		return notification.Receipt{}, sendError(notification.Rejected, "invalid_content")
	}
	httpRequest, err := newJSONRequest(ctx, resendEndpoint, body)
	if err != nil {
		return notification.Receipt{}, sendError(notification.Rejected, "invalid_request")
	}
	httpRequest.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpRequest.Header.Set("Idempotency-Key", request.DeliveryID)
	responseBody, status, headers, err := send(p.client, httpRequest)
	if err != nil {
		return notification.Receipt{}, err
	}
	if status < 200 || status >= 300 {
		return notification.Receipt{}, resendStatusError(status, headers, responseBody)
	}
	var response resendResponse
	if json.Unmarshal(responseBody, &response) != nil || response.ID == "" {
		return notification.Receipt{}, sendError(notification.Ambiguous, "invalid_response")
	}
	return notification.Receipt{MessageID: response.ID}, nil
}

func resendStatusError(status int, headers http.Header, body []byte) error {
	if status != http.StatusConflict {
		return statusError(status, headers)
	}
	var response resendErrorResponse
	if json.Unmarshal(body, &response) != nil {
		return statusError(status, headers)
	}
	switch response.Name {
	case "concurrent_idempotent_requests":
		return sendError(notification.Ambiguous, "concurrent_idempotent_requests")
	case "invalid_idempotent_request":
		return sendError(notification.Rejected, "invalid_idempotent_request")
	default:
		return statusError(status, headers)
	}
}
