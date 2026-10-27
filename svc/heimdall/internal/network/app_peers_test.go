package network

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAppPeersReadsCiliumWorkloadIdentityFilter(t *testing.T) {
	directory := t.TempDir()
	type entry struct {
		CIDR     string `json:"cidr"`
		Identity int64  `json:"identity"`
	}
	for _, tc := range []struct {
		name      string
		status    int
		entries   []entry
		want      map[netip.Addr]struct{}
		wantError bool
	}{
		{
			name: "local and remote workloads", status: http.StatusOK,
			entries: []entry{{CIDR: "10.71.0.8/32", Identity: 1001}, {CIDR: "10.72.0.9/32", Identity: 131073}, {CIDR: "fd00::9/128", Identity: 131073}, {CIDR: "10.72.0.9/32", Identity: 131073}},
			want:    map[netip.Addr]struct{}{netip.MustParseAddr("10.71.0.8"): {}, netip.MustParseAddr("10.72.0.9"): {}, netip.MustParseAddr("fd00::9"): {}},
		},
		{name: "exclude non-host and public prefixes", status: http.StatusOK, entries: []entry{{CIDR: "10.0.0.0/8", Identity: 1001}, {CIDR: "1.1.1.1/32", Identity: 1001}, {CIDR: "10.71.0.8/32", Identity: 0}}, want: map[netip.Addr]struct{}{}},
		{name: "withdrawn identities", status: http.StatusNotFound, want: map[netip.Addr]struct{}{}},
		{name: "unavailable agent", status: http.StatusServiceUnavailable, wantError: true},
		{name: "invalid address", status: http.StatusOK, entries: []entry{{CIDR: "bad", Identity: 1001}}, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body, err := json.Marshal(tc.entries)
			require.NoError(t, err)
			filters := make(chan []string, 1)
			server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/v1/ip" || r.Header.Get("Content-Type") != "application/json" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				var labels []string
				if err := json.NewDecoder(r.Body).Decode(&labels); err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				filters <- labels
				w.WriteHeader(tc.status)
				if _, err := w.Write(body); err != nil {
					t.Error(err)
				}
			}))
			require.NoError(t, server.Listener.Close())
			socket := filepath.Join(directory, "cilium.sock")
			server.Listener, err = net.Listen("unix", socket)
			require.NoError(t, err)
			server.Start()
			t.Cleanup(server.Close)
			peers, err := NewAppPeers(socket).Addresses(t.Context())
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.want, peers)
			}
			select {
			case labels := <-filters:
				require.ElementsMatch(t, []string{"k8s:app.kubernetes.io/managed-by=krane", "k8s:app.kubernetes.io/component=deployment"}, labels)
			default:
				t.Fatal("Cilium workload filter was not sent")
			}
		})
	}
}
