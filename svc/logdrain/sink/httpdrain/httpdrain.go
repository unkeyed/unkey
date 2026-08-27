// Package httpdrain delivers log batches to generic HTTPS endpoints. Each
// event is one object in the WorkOS log stream shape:
//
//	{"event":{...},"timestamp":"2024-01-15T10:30:00.123Z"}
//
// The body is either one JSON array of those objects (the default) or
// newline-delimited JSON (one object per line), selected by [Config.Format].
// Batch metadata travels in X-Unkey-* request headers.
package httpdrain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/ssrf"
	"github.com/unkeyed/unkey/svc/logdrain/sink"
)

// Body formats for [Config.Format].
const (
	// FormatNDJSON delivers one JSON object per line with
	// Content-Type application/x-ndjson.
	FormatNDJSON = "ndjson"
	// FormatJSON delivers one JSON array of event objects with
	// Content-Type application/json. This is the default.
	FormatJSON = "json"
)

// Config configures one generic HTTP drain.
type Config struct {
	// Endpoint is the customer-provided HTTPS URL that receives the body.
	Endpoint string
	// Format selects the body encoding: FormatJSON or FormatNDJSON.
	// Empty defaults to FormatJSON.
	Format string
	// Headers are customer-provided request headers, typically for authentication, sent verbatim on every delivery.
	Headers map[string]string
	// Timeout is the per-request timeout.
	Timeout time.Duration

	// UnsafeAllowTestEndpoint disables the transport safety checks so tests
	// and local development can target private, plain-http endpoints: the
	// SSRF guard that rejects endpoints resolving to loopback, private, or
	// link-local addresses, and the requirement that endpoints use https.
	UnsafeAllowTestEndpoint bool
}

// Sink delivers WorkOS-shaped event envelopes to one customer HTTP endpoint.
type Sink struct {
	cfg    Config
	client *http.Client
}

var _ sink.Sink = (*Sink)(nil)

// New validates the endpoint and format and returns an error for a forbidden
// or non-https endpoint or an unknown format.
func New(cfg Config) (*Sink, error) {
	switch cfg.Format {
	case "", FormatNDJSON, FormatJSON:
	default:
		return nil, fmt.Errorf("unknown http drain format %q", cfg.Format)
	}
	opts := []ssrf.Option{ssrf.WithTimeout(cfg.Timeout)}
	if cfg.UnsafeAllowTestEndpoint {
		opts = append(opts, ssrf.UnsafeAllowAll())
	}
	if err := ssrf.ValidateEndpoint(cfg.Endpoint, opts...); err != nil {
		return nil, err
	}
	return &Sink{cfg: cfg, client: ssrf.New(opts...)}, nil
}

// Deliver returns a result for each completed HTTP response. Only a 2xx
// response acknowledges the batch.
func (a *Sink) Deliver(ctx context.Context, batch sink.Batch) (sink.Result, error) {
	contentType := "application/json"
	if a.cfg.Format == FormatNDJSON {
		contentType = "application/x-ndjson"
	}
	body, err := marshalBatch(batch, a.cfg.Format)
	if err != nil {
		return sink.Result{}, fmt.Errorf("marshal batch: %w", err)
	}
	result := sink.Result{
		Acknowledged:     false,
		HTTPStatus:       0,
		ResponseBody:     "",
		RequestBodyBytes: int64(len(body)),
		RetryAfter:       0,
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.cfg.Endpoint, bytes.NewReader(body))
	if err != nil {
		return result, fmt.Errorf("create request: %w", err)
	}
	for name, value := range a.cfg.Headers {
		req.Header.Set(name, value)
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("User-Agent", "unkey-logdrain/1")
	req.Header.Set("X-Unkey-Schema-Version", batch.SchemaVersion)
	req.Header.Set("X-Unkey-Drain-Id", batch.DrainID)
	req.Header.Set("X-Unkey-Workspace-Id", batch.WorkspaceID)

	resp, err := a.client.Do(req)
	if err != nil {
		return result, fmt.Errorf("deliver HTTP request: %w", err)
	}
	result.HTTPStatus = resp.StatusCode
	diagnostic, err := sink.ReadDiagnostic(resp.Body)
	if err != nil {
		return result, err
	}
	retryAfter, _ := sink.ParseRetryAfter(resp.Header.Get("Retry-After"), time.Now())
	result.Acknowledged = resp.StatusCode >= 200 && resp.StatusCode < 300
	result.ResponseBody = strings.TrimSpace(string(diagnostic))
	result.RetryAfter = retryAfter
	return result, nil
}

// batchLine is one event in the WorkOS log stream shape. Batch metadata
// travels in request headers, so the body stays pure event data.
type batchLine struct {
	Event     sink.Payload `json:"event"`
	Timestamp string       `json:"timestamp"`
}

// marshalBatch encodes the events as one JSON array of event objects, or as
// one NDJSON line per event when format is FormatNDJSON.
func marshalBatch(batch sink.Batch, format string) ([]byte, error) {
	if format == FormatNDJSON {
		var body bytes.Buffer
		encoder := json.NewEncoder(&body)
		for _, event := range batch.Events {
			if err := encoder.Encode(batchLine{event.Payload, sink.FormatTime(event.Time)}); err != nil {
				return nil, err
			}
		}
		return body.Bytes(), nil
	}
	lines := make([]batchLine, len(batch.Events))
	for i, event := range batch.Events {
		lines[i] = batchLine{event.Payload, sink.FormatTime(event.Time)}
	}
	return json.Marshal(lines)
}
