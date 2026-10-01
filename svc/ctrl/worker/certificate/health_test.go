package certificate

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

var certificateMetrics = prometheus.NewRegistry()

func TestMain(m *testing.M) {
	lazy.SetRegistry(certificateMetrics)
	os.Exit(m.Run())
}

func TestHealthMetricsQueryOnlyOnDedicatedScrape(t *testing.T) {
	database := &healthDB{health: db.GetCertificateHealthRow{
		FailedChallenges: 3,
		EarliestExpiry:   1790742010123,
	}}
	svc := New(Config{DB: database})
	handler := svc.HealthMetricsHandler()
	require.Zero(t, database.calls)

	response := httptest.NewRecorder()
	promhttp.HandlerFor(certificateMetrics, promhttp.HandlerOpts{}).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	require.Equal(t, http.StatusOK, response.Code)
	require.NotContains(t, response.Body.String(), "unkey_control_certificate_failed_challenges")
	require.Zero(t, database.calls)

	for _, test := range []struct {
		name   string
		health db.GetCertificateHealthRow
		err    error
		status int
		lines  []string
	}{
		{
			name: "current state", health: database.health, status: http.StatusOK,
			lines: []string{
				"unkey_control_certificate_failed_challenges 3\n",
				"unkey_control_certificate_earliest_expiry_timestamp_seconds 1.790742010123e+09\n",
			},
		},
		{name: "database failure", err: errors.New("database unavailable"), status: http.StatusInternalServerError},
		{name: "database timeout", err: context.DeadlineExceeded, status: http.StatusInternalServerError},
		{
			name: "recovery with no certificates", status: http.StatusOK,
			lines: []string{
				"unkey_control_certificate_failed_challenges 0\n",
				"unkey_control_certificate_earliest_expiry_timestamp_seconds 0\n",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			database.health, database.err = test.health, test.err
			before := database.calls
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics/certificates", nil))
			require.Equal(t, before+1, database.calls)
			require.Equal(t, test.status, response.Code)
			require.WithinDuration(t, time.Now().Add(10*time.Second), database.deadline, time.Second)
			for _, line := range test.lines {
				require.Contains(t, response.Body.String(), "\n"+line)
			}
			if test.err != nil {
				require.Contains(t, response.Body.String(), test.err.Error())
				require.NotContains(t, response.Body.String(), "# TYPE")
			}
		})
	}

	before := database.calls
	response = httptest.NewRecorder()
	promhttp.HandlerFor(certificateMetrics, promhttp.HandlerOpts{}).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	require.Equal(t, http.StatusOK, response.Code)
	require.NotContains(t, response.Body.String(), "unkey_control_certificate_failed_challenges")
	require.Equal(t, before, database.calls)
}

type healthDB struct {
	db.Database
	health   db.GetCertificateHealthRow
	err      error
	calls    int
	deadline time.Time
}

func (d *healthDB) GetCertificateHealth(ctx context.Context) (db.GetCertificateHealthRow, error) {
	d.calls++
	if deadline, ok := ctx.Deadline(); ok {
		d.deadline = deadline
	}
	return d.health, d.err
}
