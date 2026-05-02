// Package axiom is the Axiom sink for logdrain. It POSTs JSON arrays to
// {endpoint}/v1/datasets/{dataset}/ingest with a Bearer token.
package axiom

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/svc/logdrain/internal/metrics"
	"github.com/unkeyed/unkey/svc/logdrain/internal/sinks"
)

// Config holds the non-secret half of an Axiom drain. The token is supplied
// separately so it never crosses the same struct boundary as the
// JSON-marshaled config column on log_drains.
//
// JSON tags here are part of the wire contract with the dashboard: the
// dashboard writes log_drains.config in this exact shape, and the
// coordinator unmarshals it back. Renaming Go fields is fine; renaming
// JSON keys is a migration.
type Config struct {
	// Dataset is the Axiom dataset name to ingest into.
	Dataset string `json:"dataset"`

	// Endpoint is the API root, defaulting to https://api.axiom.co. Kept
	// configurable so EU customers can point at api.eu.axiom.co without a
	// code change.
	Endpoint string `json:"endpoint,omitempty"`
}

// Sink ingests via POST {endpoint}/v1/datasets/{dataset}/ingest with a
// Bearer token. Axiom accepts JSON arrays of arbitrary objects; we project
// the Record's tenant identifiers into top-level keys plus an _attributes
// nested object for the parsed structured payload, so the dataset's
// auto-flattening surfaces both as queryable fields.
type Sink struct {
	cfg    Config
	token  string
	client *http.Client
}

// Compile-time guarantee Sink satisfies the sinks.Sink interface.
var _ sinks.Sink = (*Sink)(nil)

// New returns a sink configured to push to the given Axiom dataset. The
// default endpoint and a 30s HTTP timeout are applied when the caller
// leaves them empty — a tighter default than net/http's indefinite block,
// since logdrain blocks the cursor advance on Send.
func New(cfg Config, token string, client *http.Client) *Sink {
	if cfg.Endpoint == "" {
		cfg.Endpoint = "https://api.axiom.co"
	}
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &Sink{cfg: cfg, token: token, client: client}
}

// event is the per-record JSON shape. _time is RFC3339 nanoseconds, the
// format Axiom recognises without a custom parser hint.
type event struct {
	Time          string         `json:"_time"`
	Level         string         `json:"level,omitempty"`
	Message       string         `json:"message,omitempty"`
	WorkspaceID   string         `json:"workspace_id,omitempty"`
	ProjectID     string         `json:"project_id,omitempty"`
	EnvironmentID string         `json:"environment_id,omitempty"`
	AppID         string         `json:"app_id,omitempty"`
	DeploymentID  string         `json:"deployment_id,omitempty"`
	Region        string         `json:"region,omitempty"`
	Platform      string         `json:"platform,omitempty"`
	PodName       string         `json:"k8s_pod_name,omitempty"`
	Source        string         `json:"source,omitempty"`
	Attributes    map[string]any `json:"attributes,omitempty"`
}

// maxChunkBytes caps the *uncompressed* JSON size we ship to Axiom in a
// single HTTP request. Axiom's per-request ceiling is documented as
// "permissive multi-MB" without an exact number; we pick 1 MiB as a
// conservative chunk that fits any plan tier and stays inside the 1 MB
// figure quoted in the v1 [hardening plan]. Larger batches are split into
// multiple sequential requests.
//
// [hardening plan]: /architecture/services/logdrain/hardening-plan
const maxChunkBytes = 1 << 20

// Send marshals the batch as a JSON array, gzip-encodes it, and POSTs it
// to Axiom. Batches over maxChunkBytes (uncompressed) are split into
// multiple sequential HTTP requests so a single oversized tick can't
// trip Axiom's request-size ceiling. The first failing chunk short-
// circuits the rest; the worker's retry helper re-runs Send from the
// start and gzip determinism keeps the duplicate window bounded.
func (s *Sink) Send(ctx context.Context, batch []sinks.Record) error {
	start := time.Now()
	defer func() {
		metrics.ProviderRequestDuration.WithLabelValues("axiom").Observe(time.Since(start).Seconds())
	}()

	if len(batch) == 0 {
		return nil
	}
	encoded, err := sinks.EncodeJSON(batch, toEvent)
	if err != nil {
		return fmt.Errorf("marshal axiom batch: %w", err)
	}
	return sinks.ChunkByBytes(encoded, maxChunkBytes, 0, func(chunk [][]byte) error {
		return s.sendChunk(ctx, chunk)
	})
}

// sendChunk wraps a single chunk of pre-encoded events in `[…]` and
// gzip-POSTs it. Splitting this out keeps Send focused on chunk
// boundaries and lets the metric / error path live in one place
// regardless of how many chunks the batch was split into.
func (s *Sink) sendChunk(ctx context.Context, chunk [][]byte) error {
	if len(chunk) == 0 {
		return nil
	}

	gz, err := sinks.GzipBytes(sinks.BuildJSONArray(chunk))
	if err != nil {
		return fmt.Errorf("gzip axiom chunk: %w", err)
	}

	url := fmt.Sprintf("%s/v1/datasets/%s/ingest", s.cfg.Endpoint, s.cfg.Dataset)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(gz))
	if err != nil {
		return fmt.Errorf("build axiom request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	resp, err := s.client.Do(req)
	if err != nil {
		metrics.ProviderErrors.WithLabelValues("axiom", classifyError(err)).Inc()
		return sinks.TransportError("axiom", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		// Aggregated metrics only — per-workspace counters dropped per
		// the cardinality cleanup. Track gzip-shipped bytes since that's
		// what actually leaves the pod.
		metrics.RecordsDelivered.WithLabelValues("axiom").Add(float64(len(chunk)))
		metrics.RecordsBytesDelivered.WithLabelValues("axiom").Add(float64(len(gz)))

		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}

	errorType := classifyHTTPError(resp.StatusCode)
	metrics.ProviderErrors.WithLabelValues("axiom", errorType).Inc()

	msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return sinks.HTTPError("axiom", resp, msg)
}

// HealthCheck pushes a single synthetic record through the live ingest
// path so dashboard test-push surfaces real auth or dataset errors at
// drain creation time.
func (s *Sink) HealthCheck(ctx context.Context) error {
	return s.Send(ctx, []sinks.Record{sinks.HealthCheckRecord()})
}

func toEvent(r sinks.Record) event {
	return event{
		Time:          time.UnixMilli(r.TimeMs).UTC().Format(time.RFC3339Nano),
		Level:         r.SeverityText,
		Message:       r.Body,
		WorkspaceID:   r.WorkspaceID,
		ProjectID:     r.ProjectID,
		EnvironmentID: r.EnvironmentID,
		AppID:         r.AppID,
		DeploymentID:  r.DeploymentID,
		Region:        r.Region,
		Platform:      r.Platform,
		PodName:       r.K8sPodName,
		Source:        sinks.SourceLabel(r.Kind),
		Attributes:    r.Attributes,
	}
}

// classifyError categorizes errors for metrics tracking
func classifyError(err error) string {
	// TODO: Implement more sophisticated error classification
	return "network"
}

// classifyHTTPError categorizes HTTP status codes for metrics
func classifyHTTPError(statusCode int) string {
	switch {
	case statusCode == 401 || statusCode == 403:
		return "auth"
	case statusCode == 429:
		return "rate_limit"
	case statusCode >= 400 && statusCode < 500:
		return "4xx"
	case statusCode >= 500:
		return "5xx"
	default:
		return "unknown"
	}
}
