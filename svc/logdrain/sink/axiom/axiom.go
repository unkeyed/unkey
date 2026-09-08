// Package axiom delivers log batches to Axiom's dataset ingest API as NDJSON.
package axiom

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/ssrf"
	"github.com/unkeyed/unkey/svc/logdrain/sink"
)

const apiBaseURL = "https://api.axiom.co"

// Config configures one Axiom dataset destination.
type Config struct {
	// Dataset identifies the Axiom dataset that receives events.
	Dataset string
	// Token authenticates requests to Axiom.
	Token string
	// Timeout bounds each request to Axiom.
	Timeout time.Duration
}

// Sink delivers event batches to one Axiom dataset.
type Sink struct {
	endpoint string
	token    string
	client   *http.Client
}

var _ sink.Sink = (*Sink)(nil)

// New validates destination configuration and returns an error for a missing
// dataset, missing token, or forbidden or non-https endpoint.
func New(cfg Config) (*Sink, error) {
	if err := assert.All(
		assert.NotEmpty(cfg.Dataset, "axiom dataset is required"),
		assert.NotEmpty(cfg.Token, "axiom token is required"),
	); err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("%s/v1/datasets/%s/ingest", apiBaseURL, url.PathEscape(cfg.Dataset))
	opts := []ssrf.Option{ssrf.WithTimeout(cfg.Timeout)}
	if err := ssrf.ValidateEndpoint(endpoint, opts...); err != nil {
		return nil, err
	}
	return &Sink{endpoint: endpoint, token: cfg.Token, client: ssrf.New(opts...)}, nil
}

// Deliver acknowledges the batch only after Axiom accepts every event.
func (a *Sink) Deliver(ctx context.Context, batch sink.Batch) (sink.Result, error) {
	body, err := marshalEvents(batch.Events)
	if err != nil {
		return sink.Result{}, fmt.Errorf("marshal events: %w", err)
	}
	delivery := sink.Result{
		Acknowledged:     false,
		HTTPStatus:       0,
		ResponseBody:     "",
		RequestBodyBytes: int64(len(body)),
		RetryAfter:       0,
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.endpoint, bytes.NewReader(body))
	if err != nil {
		return delivery, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+a.token)
	req.Header.Set("Content-Type", "application/x-ndjson")
	req.Header.Set("User-Agent", "unkey-logdrain/1")
	resp, err := a.client.Do(req)
	if err != nil {
		return delivery, fmt.Errorf("deliver Axiom request: %w", err)
	}
	delivery.HTTPStatus = resp.StatusCode
	diagnostic, err := sink.ReadDiagnostic(resp.Body)
	if err != nil {
		return delivery, err
	}
	delivery.Acknowledged = resp.StatusCode >= 200 && resp.StatusCode < 300
	delivery.ResponseBody = strings.TrimSpace(string(diagnostic))
	delivery.RetryAfter = retryAfter(resp.Header, time.Now())
	if !delivery.Acknowledged {
		return delivery, nil
	}
	var ingestion struct {
		Failed int `json:"failed"`
	}
	if json.Unmarshal(diagnostic, &ingestion) == nil && ingestion.Failed > 0 {
		delivery.Acknowledged = false
	}
	return delivery, nil
}

// retryAfter applies Axiom's response-header precedence. Retry-After uses the
// standard format. X-RateLimit-Reset contains a UTC Unix timestamp in seconds.
func retryAfter(headers http.Header, now time.Time) time.Duration {
	if delay, ok := sink.ParseRetryAfter(headers.Get("Retry-After"), now); ok {
		return delay
	}
	resetAt, err := strconv.ParseInt(strings.TrimSpace(headers.Get("X-RateLimit-Reset")), 10, 64)
	if err != nil {
		return 0
	}
	delay := time.Unix(resetAt, 0).Sub(now)
	if delay <= 0 {
		return 0
	}
	return delay
}

// marshalEvents produces one Axiom-compatible NDJSON line per event.
func marshalEvents(events []sink.Event) ([]byte, error) {
	var body bytes.Buffer
	encoder := json.NewEncoder(&body)
	for _, event := range events {
		line := struct {
			Timestamp string       `json:"_time"`
			Stream    string       `json:"stream"`
			Event     sink.Payload `json:"event"`
		}{sink.FormatTime(event.Time), event.Stream, event.Payload}
		if err := encoder.Encode(line); err != nil {
			return nil, err
		}
	}
	return body.Bytes(), nil
}
