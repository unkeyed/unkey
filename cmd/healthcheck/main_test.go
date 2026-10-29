package healthcheck_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/healthcheck"
)

func TestHealthcheckReportsTheHTTPResult(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusServiceUnavailable} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("unexpected method: %s", r.Method)
				}
				w.WriteHeader(status)
			}))
			t.Cleanup(server.Close)
			file, err := os.CreateTemp(t.TempDir(), "stdout")
			require.NoError(t, err)
			stdout := os.Stdout
			os.Stdout = file
			t.Cleanup(func() {
				os.Stdout = stdout
				require.NoError(t, file.Close())
			})
			err = healthcheck.Cmd.Run(context.Background(), []string{"unkey healthcheck", server.URL})
			if status == http.StatusOK {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, fmt.Sprintf("healthcheck failed: HTTP %d from %s", status, server.URL))
			}
			_, err = file.Seek(0, io.SeekStart)
			require.NoError(t, err)
			out, err := io.ReadAll(file)
			require.NoError(t, err)
			if status == http.StatusOK {
				require.Equal(t, "Healthy: HTTP 200 from "+server.URL+"\n", string(out))
			} else {
				require.Empty(t, out)
			}
		})
	}
	for _, args := range [][]string{nil, {""}, {"http://example.com", "unexpected"}} {
		err := healthcheck.Cmd.Run(context.Background(), append([]string{"unkey healthcheck"}, args...))
		require.EqualError(t, err, "usage: unkey healthcheck <url>")
	}
}
