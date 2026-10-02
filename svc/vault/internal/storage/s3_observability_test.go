package storage_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/logger/loggertest"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
	"github.com/unkeyed/unkey/svc/vault/internal/storage"
)

var registry = prometheus.NewRegistry()

func TestMain(m *testing.M) {
	lazy.SetRegistry(registry)
	os.Exit(m.Run())
}

func TestS3_LogsAndCountsInvalidCredentials(t *testing.T) {
	store := s3Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusForbidden)
		_, err := fmt.Fprint(w, `<Error><Code>InvalidAccessKeyId</Code><Message>sensitive-provider-message</Message></Error>`)
		if err != nil {
			t.Error(err)
		}
	}))
	logs := loggertest.Install(t)
	failures := metricValue(t, "get", "error", "InvalidAccessKeyId")

	data, found, err := store.GetObject(t.Context(), "sensitive-keyring/dek")
	require.Error(t, err)
	require.False(t, found)
	require.Nil(t, data)
	require.Equal(t, failures+1, metricValue(t, "get", "error", "InvalidAccessKeyId"))
	record := logs.Find(t, "vault s3 operation failed")
	require.Equal(t, slog.LevelError, record.Level)
	require.Equal(t, map[string]any{
		"operation": "get", "error_code": "InvalidAccessKeyId", "http_status": int64(403),
	}, loggertest.FlatAttrs(record))
	require.Len(t, logs.Records(), 1)
}

func TestS3_ReportsAccessLossAndRecovery(t *testing.T) {
	for _, tt := range []struct{ operation, code string }{
		{"get", "SignatureDoesNotMatch"},
		{"put", "AccessDenied"},
		{"list", "ExpiredToken"},
	} {
		t.Run(tt.operation, func(t *testing.T) {
			var denied atomic.Bool
			store := s3Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if denied.Load() {
					w.Header().Set("Content-Type", "application/xml")
					w.WriteHeader(http.StatusForbidden)
					_, err := fmt.Fprintf(w, `<Error><Code>%s</Code><Message>sensitive-provider-message</Message></Error>`, tt.code)
					if err != nil {
						t.Error(err)
					}
					return
				}
				body := ""
				switch tt.operation {
				case "get":
					body = "sensitive-encrypted-dek"
				case "list":
					body = `<ListBucketResult><Contents><Key>sensitive-keyring/dek</Key></Contents></ListBucketResult>`
				}
				_, err := fmt.Fprint(w, body)
				if err != nil {
					t.Error(err)
				}
			}))
			logs := loggertest.Install(t)
			successes := metricValue(t, tt.operation, "success", "")
			failures := metricValue(t, tt.operation, "error", tt.code)
			call := func() error {
				switch tt.operation {
				case "put":
					return store.PutObject(t.Context(), "sensitive-keyring/dek", []byte("sensitive-encrypted-dek"))
				case "list":
					keys, err := store.ListObjectKeys(t.Context(), "sensitive-keyring/")
					if err == nil {
						require.Equal(t, []string{"sensitive-keyring/dek"}, keys)
					}
					return err
				default:
					data, found, err := store.GetObject(t.Context(), "sensitive-keyring/dek")
					if err == nil {
						require.True(t, found)
						require.Equal(t, []byte("sensitive-encrypted-dek"), data)
					}
					return err
				}
			}
			require.NoError(t, call())
			denied.Store(true)
			require.Error(t, call())
			require.Equal(t, failures+1, metricValue(t, tt.operation, "error", tt.code))
			record := logs.Find(t, "vault s3 operation failed")
			require.Equal(t, slog.LevelError, record.Level)
			require.Equal(t, map[string]any{
				"operation": tt.operation, "error_code": tt.code, "http_status": int64(403),
			}, loggertest.FlatAttrs(record))
			denied.Store(false)
			require.NoError(t, call())
			require.Equal(t, successes+2, metricValue(t, tt.operation, "success", ""))
			require.Len(t, logs.Records(), 1)
		})
	}
}

func TestS3_PreservesNotFoundHandling(t *testing.T) {
	for _, tt := range []struct {
		code, outcome, metricCode string
		status                    int
	}{
		{"NoSuchBucket", "success", "", http.StatusNotFound},
		{"NoSuchKey", "success", "", http.StatusNotFound},
		{"", "success", "", http.StatusNotFound},
		{"sensitive-unrecognized-code", "success", "", http.StatusNotFound},
		{"sensitive-unrecognized-code", "error", "other", http.StatusForbidden},
	} {
		t.Run(fmt.Sprintf("%s/%d", tt.code, tt.status), func(t *testing.T) {
			store := s3Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/xml")
				w.WriteHeader(tt.status)
				if tt.code != "" {
					_, err := fmt.Fprintf(w, `<Error><Code>%s</Code><Message>sensitive-provider-message</Message></Error>`, tt.code)
					if err != nil {
						t.Error(err)
					}
				}
			}))
			logs := loggertest.Install(t)
			before := metricValue(t, "get", tt.outcome, tt.metricCode)
			data, found, err := store.GetObject(t.Context(), "sensitive-keyring/LATEST")
			if tt.outcome == "error" {
				require.Error(t, err)
				require.Len(t, logs.Records(), 1)
				record := logs.Find(t, "vault s3 operation failed")
				require.Equal(t, map[string]any{
					"operation": "get", "error_code": tt.metricCode, "http_status": int64(tt.status),
				}, loggertest.FlatAttrs(record))
			} else {
				require.NoError(t, err)
				require.Empty(t, logs.Records())
			}
			require.False(t, found)
			require.Nil(t, data)
			require.Equal(t, before+1, metricValue(t, "get", tt.outcome, tt.metricCode))
			require.NotContains(t, scrape(t), "sensitive-")
		})
	}
}

func TestS3_DistinguishesCancellationFromTimeout(t *testing.T) {
	store := s3Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("an already canceled request must not reach S3")
	}))
	logs := loggertest.Install(t)
	canceled := metricValue(t, "get", "canceled", "")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, _, err := store.GetObject(ctx, "sensitive-keyring/dek")
	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, logs.Records())
	require.Equal(t, canceled+1, metricValue(t, "get", "canceled", ""))

	timeouts := metricValue(t, "get", "error", "timeout")
	ctx, cancel = context.WithDeadline(t.Context(), time.Unix(1, 0))
	t.Cleanup(cancel)
	_, _, err = store.GetObject(ctx, "sensitive-keyring/dek")
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Equal(t, timeouts+1, metricValue(t, "get", "error", "timeout"))
	require.Len(t, logs.Records(), 1)
	record := logs.Find(t, "vault s3 operation failed")
	require.Equal(t, map[string]any{
		"operation": "get", "error_code": "timeout", "http_status": int64(0),
	}, loggertest.FlatAttrs(record))
}

func TestS3_ReportsFinalStorageFailures(t *testing.T) {
	for _, tt := range []struct {
		name                 string
		status, loggedStatus int
		attempts             int32
	}{
		{"after SDK retries", http.StatusServiceUnavailable, http.StatusServiceUnavailable, 3},
		{"after successful headers but incomplete body", http.StatusOK, 0, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var attempts atomic.Int32
			store := s3Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attempts.Add(1)
				w.Header().Set("Content-Length", "1000")
				w.WriteHeader(tt.status)
				_, err := fmt.Fprint(w, "sensitive-incomplete-payload")
				if err != nil {
					t.Error(err)
				}
			}))
			logs := loggertest.Install(t)
			failures := metricValue(t, "get", "error", "other")
			successes := metricValue(t, "get", "success", "")
			data, found, err := store.GetObject(t.Context(), "sensitive-keyring/dek")
			require.Error(t, err)
			require.Nil(t, data)
			require.False(t, found)
			require.Equal(t, tt.attempts, attempts.Load())
			require.Equal(t, failures+1, metricValue(t, "get", "error", "other"))
			require.Equal(t, successes, metricValue(t, "get", "success", ""))
			require.Len(t, logs.Records(), 1)
			record := logs.Find(t, "vault s3 operation failed")
			require.Equal(t, map[string]any{
				"operation": "get", "error_code": "other", "http_status": int64(tt.loggedStatus),
			}, loggertest.FlatAttrs(record))
		})
	}
}

func s3Server(t *testing.T, handler http.Handler) storage.Storage {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	store, err := storage.NewS3(storage.S3Config{
		S3URL: server.URL, S3Bucket: "vault",
		S3AccessKeyID: "sensitive-access-key", S3AccessKeySecret: "sensitive-secret-key",
	})
	require.NoError(t, err)
	return store
}

func scrape(t *testing.T) string {
	t.Helper()
	w := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	require.Equal(t, http.StatusOK, w.Code)
	return w.Body.String()
}

func metricValue(t *testing.T, operation, outcome, code string) float64 {
	t.Helper()
	prefix := fmt.Sprintf(`unkey_vault_s3_operations_total{error_code=%q,operation=%q,outcome=%q} `, code, operation, outcome)
	for line := range strings.SplitSeq(scrape(t), "\n") {
		if value, ok := strings.CutPrefix(line, prefix); ok {
			n, err := strconv.ParseFloat(value, 64)
			require.NoError(t, err)
			return n
		}
	}
	return 0
}
