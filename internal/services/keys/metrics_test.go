package keys

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/require"
	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

var testRegistry = prometheus.NewRegistry()

func TestMain(m *testing.M) {
	lazy.SetRegistry(testRegistry)
	os.Exit(m.Run())
}

func TestRecordVerificationAttributesKeyWorkspace(t *testing.T) {
	t.Parallel()

	kv := &KeyVerifier{
		Status: StatusDisabled,
		Key:    keysdb.FindKeyForVerificationRow{WorkspaceID: "ws_metric_attr_test"},
	}

	before := verificationCount(t, "key", "DISABLED", "ws_metric_attr_test")
	recordVerification(kv)
	require.Equal(t, before+1, verificationCount(t, "key", "DISABLED", "ws_metric_attr_test"))
}

func TestRecordVerificationWithoutKeyHasEmptyWorkspace(t *testing.T) {
	t.Parallel()

	kv := &KeyVerifier{Status: StatusNotFound}

	before := verificationCount(t, "key", "NOT_FOUND", "")
	recordVerification(kv)
	require.Equal(t, before+1, verificationCount(t, "key", "NOT_FOUND", ""))
}

func TestRecordVerificationIgnoresNilVerifier(t *testing.T) {
	t.Parallel()

	recordVerification(nil)
}

func verificationCount(t *testing.T, keyType, code, workspaceID string) float64 {
	t.Helper()

	w := httptest.NewRecorder()
	promhttp.HandlerFor(testRegistry, promhttp.HandlerOpts{}).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	require.Equal(t, http.StatusOK, w.Code)

	prefix := fmt.Sprintf("unkey_key_verifications_total{code=%q,type=%q,workspace_id=%q} ", code, keyType, workspaceID)
	for line := range strings.SplitSeq(w.Body.String(), "\n") {
		if value, ok := strings.CutPrefix(line, prefix); ok {
			n, err := strconv.ParseFloat(value, 64)
			require.NoError(t, err)
			return n
		}
	}
	return 0
}
