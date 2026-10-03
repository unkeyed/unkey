package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/unkeyed/unkey/pkg/cli"
)

var relayCmd = &cli.Command{
	Name:  "relay-events",
	Usage: "Pull registered GitHub events from the install relay into the local control API",
	Flags: []cli.Flag{
		cli.String("relay-url", "HTTPS origin of the installation relay", cli.EnvVar("GITHUB_INSTALL_RELAY_URL"), cli.Required()),
		cli.String("origin", "Exact registered dashboard HTTPS origin", cli.EnvVar("DASHBOARD_BASE_URL"), cli.Required()),
		cli.String("webhook-url", "Loopback control API webhook endpoint", cli.Default("http://localhost:7091/webhooks/github")),
	},
	Action: relayEvents,
}

const relayResponseLimit = 13 << 20

var relayHandle = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)

type eventForwarder struct {
	client     *http.Client
	relayURL   string
	origin     string
	webhookURL string
	adminToken string
	token      string
	expiresAt  time.Time
	enabled    bool
}

type relayDelivery struct {
	ID         string `json:"id"`
	LeaseToken string `json:"leaseToken"`
	Event      string `json:"event"`
	DeliveryID string `json:"deliveryId"`
	Signature  string `json:"signature"`
	Payload    string `json:"payload"`
}

func relayEvents(ctx context.Context, cmd *cli.Command) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	f, err := newEventForwarder(cmd.String("relay-url"), cmd.String("origin"), cmd.String("webhook-url"), os.Getenv("GITHUB_INSTALL_RELAY_TOKEN"), os.Getenv("GITHUB_INSTALL_RELAY_ADMIN_TOKEN"))
	if err != nil {
		return err
	}
	slog.Info("GitHub relay event forwarding started", "origin", f.origin)
	for ctx.Err() == nil {
		delivered, err := f.forwardNext(ctx)
		if ctx.Err() != nil {
			return nil
		}
		if err != nil {
			slog.Warn("GitHub relay delivery failed; unacknowledged events remain queued", "error", err)
		}
		if delivered {
			continue
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(5 * time.Second):
		}
	}
	return nil
}

func newEventForwarder(relayURL, origin, webhookURL, token, adminToken string) (*eventForwarder, error) {
	for _, value := range []string{relayURL, origin} {
		u, err := url.Parse(value)
		if err != nil || u.Hostname() == "" || u.User != nil || value != "https://"+u.Host || strings.Contains(u.Host, "*") {
			return nil, errors.New("relay URL and dashboard origin must be exact HTTPS origins without a trailing slash")
		}
	}
	u, err := url.Parse(webhookURL)
	if err != nil {
		return nil, errors.New("invalid local webhook URL")
	}
	if (u.Scheme != "http" && u.Scheme != "https") || (u.Hostname() != "localhost" && !net.ParseIP(u.Hostname()).IsLoopback()) || u.User != nil || u.Path != "/webhooks/github" || u.RawQuery != "" || u.Fragment != "" || u.ForceQuery {
		return nil, errors.New("webhook URL must point to /webhooks/github on loopback")
	}
	if (token == "") == (adminToken == "") {
		return nil, errors.New("set exactly one of GITHUB_INSTALL_RELAY_TOKEN and GITHUB_INSTALL_RELAY_ADMIN_TOKEN")
	}
	if token != "" && !relayHandle.MatchString(token) {
		return nil, errors.New("invalid scoped relay token")
	}
	return &eventForwarder{
		client: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		relayURL:   relayURL,
		origin:     origin,
		webhookURL: webhookURL,
		adminToken: adminToken,
		token:      token,
		expiresAt:  time.Time{},
		enabled:    false,
	}, nil
}

func (f *eventForwarder) enroll(ctx context.Context) error {
	if f.adminToken == "" || time.Until(f.expiresAt) > 12*time.Hour {
		return nil
	}
	status, body, err := f.post(ctx, f.relayURL+"/v1/environments", f.adminToken, map[string]any{
		"origin": f.origin, "expiresAt": time.Now().Add(24 * time.Hour).UnixMilli(), "reuse": true,
	})
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("relay enrollment returned HTTP %d", status)
	}
	var enrollment struct {
		Token     string `json:"token"`
		Origin    string `json:"origin"`
		ExpiresAt int64  `json:"expiresAt"`
	}
	if err := json.Unmarshal(body, &enrollment); err != nil {
		return errors.New("invalid relay enrollment response")
	}
	expiresAt := time.UnixMilli(enrollment.ExpiresAt)
	if enrollment.Origin != f.origin || !relayHandle.MatchString(enrollment.Token) || !expiresAt.After(time.Now()) || expiresAt.After(time.Now().Add(30*24*time.Hour)) {
		return errors.New("relay enrollment does not match this environment")
	}
	if f.token != enrollment.Token {
		f.enabled = false
	}
	f.token = enrollment.Token
	f.expiresAt = expiresAt
	return nil
}

func (f *eventForwarder) forwardNext(ctx context.Context) (bool, error) {
	if err := f.enroll(ctx); err != nil {
		return false, err
	}
	if !f.enabled {
		status, _, err := f.post(ctx, f.relayURL+"/v1/webhooks/enable", f.token, struct{}{})
		if err != nil {
			return false, err
		}
		if status != http.StatusNoContent {
			return false, fmt.Errorf("relay event enrollment returned HTTP %d", status)
		}
		f.enabled = true
	}
	status, body, err := f.post(ctx, f.relayURL+"/v1/webhooks/claim", f.token, struct{}{})
	if err != nil {
		return false, err
	}
	if status == http.StatusNoContent {
		return false, nil
	}
	if status != http.StatusOK {
		return false, fmt.Errorf("relay claim returned HTTP %d", status)
	}
	var delivery relayDelivery
	if err := json.Unmarshal(body, &delivery); err != nil {
		return false, errors.New("invalid relay delivery response")
	}
	if !relayHandle.MatchString(delivery.ID) || !relayHandle.MatchString(delivery.LeaseToken) || delivery.DeliveryID == "" || delivery.Signature == "" || (delivery.Event != "push" && delivery.Event != "pull_request") || delivery.Payload == "" {
		return false, errors.New("incomplete relay delivery")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.webhookURL, strings.NewReader(delivery.Payload))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", delivery.Event)
	req.Header.Set("X-GitHub-Delivery", delivery.DeliveryID)
	req.Header.Set("X-Hub-Signature-256", delivery.Signature)
	status, _, err = f.send(req)
	if err != nil {
		return false, err
	}
	if status < 200 || status >= 300 {
		return false, fmt.Errorf("local webhook receiver returned HTTP %d; check its webhook secret and readiness", status)
	}
	status, _, err = f.post(ctx, f.relayURL+"/v1/webhooks/"+url.PathEscape(delivery.ID)+"/ack", f.token, map[string]string{"leaseToken": delivery.LeaseToken})
	if err != nil {
		return false, err
	}
	if status != http.StatusNoContent {
		return false, fmt.Errorf("relay acknowledgment returned HTTP %d", status)
	}
	slog.Info("GitHub event forwarded", "event", delivery.Event, "delivery_id", delivery.DeliveryID)
	return true, nil
}

func (f *eventForwarder) post(ctx context.Context, target, token string, value any) (int, []byte, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return 0, nil, errors.New("unable to encode relay request")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		return 0, nil, errors.New("invalid relay request URL")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	return f.send(req)
}

func (f *eventForwarder) send(req *http.Request) (int, []byte, error) {
	response, err := f.client.Do(req)
	if err != nil {
		return 0, nil, errors.New("webhook forwarding HTTP request failed")
	}
	body, readErr := io.ReadAll(io.LimitReader(response.Body, relayResponseLimit+1))
	closeErr := response.Body.Close()
	if readErr != nil || closeErr != nil {
		return 0, nil, errors.New("unable to read webhook forwarding HTTP response")
	}
	if len(body) > relayResponseLimit {
		return 0, nil, errors.New("webhook forwarding HTTP response exceeds limit")
	}
	return response.StatusCode, body, nil
}
