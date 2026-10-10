package prometheus

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

// TestNewServiceRegistry guarantees that a service registry exposes the
// runtime collectors and every lazy metric used after it is created, so a
// service that only calls NewServiceRegistry serves its own metrics.
func TestNewServiceRegistry(t *testing.T) {
	reg := NewServiceRegistry()
	probe := lazy.NewCounter(prometheus.CounterOpts{
		Namespace: "unkey",
		Subsystem: "test",
		Name:      "service_registry_total",
		Help:      "Counter used to verify lazy registration.",
	})
	probe.Inc()

	families, err := reg.Gather()
	require.NoError(t, err)
	names := make(map[string]bool, len(families))
	for _, family := range families {
		names[family.GetName()] = true
	}
	for _, name := range []string{"go_goroutines", "process_cpu_seconds_total", "unkey_test_service_registry_total"} {
		require.True(t, names[name], "registry is missing %s", name)
	}

	recorder := httptest.NewRecorder()
	Handler(reg).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "unkey_test_service_registry_total 1\n")
}
