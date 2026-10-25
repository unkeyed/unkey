package providers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/unkeyed/unkey/pkg/notification"
)

const signaturePrefix = "v1="

var reservedHeaders = map[string]struct{}{
	"Connection":          {},
	"Content-Length":      {},
	"Content-Type":        {},
	"Host":                {},
	"Proxy-Authorization": {},
	"Proxy-Connection":    {},
	"Te":                  {},
	"Trailer":             {},
	"Transfer-Encoding":   {},
	"Unkey-Delivery-Id":   {},
	"Unkey-Signature":     {},
	"Unkey-Timestamp":     {},
	"Upgrade":             {},
}

type webhook struct {
	client *http.Client
}

type webhookPayload struct {
	DeliveryID string               `json:"deliveryId"`
	Event      notification.Event   `json:"event"`
	Content    notification.Content `json:"content"`
}

func NewWebhook() notification.Provider { return &webhook{client: newHTTPClient()} }

func (p *webhook) Name() string { return "webhook" }

func (p *webhook) Channel() notification.Channel { return notification.Webhook }

func (p *webhook) IdempotencyWindow() time.Duration { return 0 }

func (p *webhook) Send(ctx context.Context, request notification.Request) (notification.Receipt, error) {
	endpoint, err := validateURL(request.Target.URL)
	if err != nil {
		return notification.Receipt{}, sendError(notification.Rejected, "invalid_webhook_url")
	}
	if request.DeliveryID == "" || strings.ContainsAny(request.DeliveryID, "\r\n") || request.Target.SigningSecret == "" {
		return notification.Receipt{}, sendError(notification.Rejected, "invalid_webhook_credentials")
	}
	if err := validateHeaders(request.Target.Headers); err != nil {
		return notification.Receipt{}, err
	}
	body, err := json.Marshal(webhookPayload{
		DeliveryID: request.DeliveryID,
		Event:      request.Event,
		Content:    request.Content,
	})
	if err != nil {
		return notification.Receipt{}, sendError(notification.Rejected, "invalid_content")
	}
	timestamp := time.Now().UTC().Format(time.RFC3339)
	mac := hmac.New(sha256.New, []byte(request.Target.SigningSecret))
	if _, err := mac.Write(append([]byte(timestamp+"."), body...)); err != nil {
		return notification.Receipt{}, sendError(notification.Rejected, "signature_failed")
	}
	httpRequest, err := newJSONRequest(ctx, endpoint.String(), body)
	if err != nil {
		return notification.Receipt{}, sendError(notification.Rejected, "invalid_request")
	}
	for name, value := range request.Target.Headers {
		httpRequest.Header.Set(name, value)
	}
	httpRequest.Header.Set("Unkey-Delivery-Id", request.DeliveryID)
	httpRequest.Header.Set("Unkey-Timestamp", timestamp)
	httpRequest.Header.Set("Unkey-Signature", signaturePrefix+hex.EncodeToString(mac.Sum(nil)))
	_, status, headers, err := send(p.client, httpRequest)
	if err != nil {
		return notification.Receipt{}, err
	}
	if status < 200 || status >= 300 {
		return notification.Receipt{}, statusError(status, headers)
	}
	return notification.Receipt{MessageID: ""}, nil
}

func validateHeaders(headers map[string]string) error {
	for name, value := range headers {
		canonical := http.CanonicalHeaderKey(name)
		if canonical == "" || !validHeaderName(name) || !validHeaderValue(value) {
			return sendError(notification.Rejected, "invalid_webhook_header")
		}
		if _, found := reservedHeaders[canonical]; found || strings.HasPrefix(canonical, "Unkey-") {
			return sendError(notification.Rejected, "reserved_webhook_header")
		}
	}
	return nil
}

func validHeaderValue(value string) bool {
	for _, character := range value {
		if character == '\t' {
			continue
		}
		if character < ' ' || character == 0x7f {
			return false
		}
	}
	return true
}

func validHeaderName(name string) bool {
	if name == "" {
		return false
	}
	for _, character := range name {
		if character > unicode.MaxASCII || !(unicode.IsLetter(character) || unicode.IsDigit(character) || strings.ContainsRune("!#$%&'*+-.^_`|~", character)) {
			return false
		}
	}
	return true
}
