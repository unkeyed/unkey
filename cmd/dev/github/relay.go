package github

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"mime"
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
	Usage: "Stream registered GitHub events from the install relay into the local control API",
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
	backoff := time.Second
	for ctx.Err() == nil {
		started := time.Now()
		err := f.forwardStream(ctx)
		if ctx.Err() != nil {
			return nil
		}
		if err != nil {
			slog.Warn("GitHub relay stream disconnected; unacknowledged events remain queued", "error", err)
		}
		if time.Since(started) >= time.Minute {
			backoff = time.Second
		}
		delay := backoff + time.Duration(rand.Int64N(int64(backoff)))
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(delay):
		}
		backoff = min(backoff*2, 30*time.Second)
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

func (f *eventForwarder) forwardStream(ctx context.Context) error {
	if err := f.enroll(ctx); err != nil {
		return err
	}
	if !f.enabled {
		status, _, err := f.post(ctx, f.relayURL+"/v1/webhooks/enable", f.token, struct{}{})
		if err != nil {
			return err
		}
		if status != http.StatusNoContent {
			return fmt.Errorf("relay event enrollment returned HTTP %d", status)
		}
		f.enabled = true
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	idle := time.AfterFunc(45*time.Second, cancel)
	defer idle.Stop()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.relayURL+"/v1/webhooks/stream", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+f.token)
	req.Header.Set("Accept", "text/event-stream")
	client := *f.client
	client.Timeout = 0
	response, err := client.Do(req)
	if err != nil {
		return errors.New("unable to open relay event stream")
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			slog.Debug("Unable to close relay event stream")
		}
	}()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("relay stream returned HTTP %d", response.StatusCode)
	}
	contentType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || contentType != "text/event-stream" {
		return errors.New("invalid relay stream content type")
	}
	scanner := bufio.NewScanner(response.Body)
	scanner.Buffer(make([]byte, 4096), relayResponseLimit)
	var data strings.Builder
	var event string
	size := 0
	for scanner.Scan() {
		idle.Reset(45 * time.Second)
		line := scanner.Text()
		size += len(line) + 1
		if size > relayResponseLimit {
			return errors.New("relay stream event exceeds limit")
		}
		if line == "" {
			if event == "delivery" {
				if err := f.forwardDelivery(ctx, data.String()); err != nil {
					return err
				}
			}
			data.Reset()
			event = ""
			size = 0
			continue
		}
		field, value, _ := strings.Cut(line, ":")
		value = strings.TrimPrefix(value, " ")
		switch field {
		case "event":
			event = value
		case "data":
			data.WriteString(value)
			data.WriteByte('\n')
		}
	}
	if scanner.Err() != nil {
		return errors.New("unable to read relay event stream")
	}
	return io.EOF
}

func (f *eventForwarder) forwardDelivery(ctx context.Context, body string) error {
	var delivery relayDelivery
	if err := json.Unmarshal([]byte(body), &delivery); err != nil {
		return errors.New("invalid relay delivery response")
	}
	if !relayHandle.MatchString(delivery.ID) || !relayHandle.MatchString(delivery.LeaseToken) || delivery.DeliveryID == "" || delivery.Signature == "" || (delivery.Event != "push" && delivery.Event != "pull_request") || delivery.Payload == "" {
		return errors.New("incomplete relay delivery")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.webhookURL, strings.NewReader(delivery.Payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", delivery.Event)
	req.Header.Set("X-GitHub-Delivery", delivery.DeliveryID)
	req.Header.Set("X-Hub-Signature-256", delivery.Signature)
	status, _, err := f.send(req)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("local webhook receiver returned HTTP %d; check its webhook secret and readiness", status)
	}
	status, _, err = f.post(ctx, f.relayURL+"/v1/webhooks/"+url.PathEscape(delivery.ID)+"/ack", f.token, map[string]string{"leaseToken": delivery.LeaseToken})
	if err != nil {
		return err
	}
	if status != http.StatusNoContent {
		return fmt.Errorf("relay acknowledgment returned HTTP %d", status)
	}
	slog.Info("GitHub event forwarded", "event", delivery.Event, "delivery_id", delivery.DeliveryID)
	return nil
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
