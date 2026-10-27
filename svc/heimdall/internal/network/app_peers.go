package network

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"time"
)

// AppPeers reads workload identities from Cilium, independent of DNS readiness.
type AppPeers struct {
	client *http.Client
}

// NewAppPeers connects to the local Cilium agent's Unix socket.
func NewAppPeers(socket string) *AppPeers {
	return &AppPeers{client: &http.Client{
		Timeout: 2 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var dialer net.Dialer
				return dialer.DialContext(ctx, "unix", socket)
			},
		},
	}}
}

// Addresses includes local and ClusterMesh workloads while they retain their
// Cilium identity, including readiness failures and graceful termination.
func (p *AppPeers) Addresses(ctx context.Context) (map[netip.Addr]struct{}, error) {
	filter, err := json.Marshal([]string{
		"k8s:app.kubernetes.io/managed-by=krane",
		"k8s:app.kubernetes.io/component=deployment",
	})
	if err != nil {
		return nil, fmt.Errorf("encode app identity filter: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://cilium/v1/ip", bytes.NewReader(filter))
	if err != nil {
		return nil, fmt.Errorf("create app identity request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("read Cilium app identities: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return map[netip.Addr]struct{}{}, nil
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("read Cilium app identities: HTTP %d", res.StatusCode)
	}

	var entries []struct {
		CIDR     string `json:"cidr"`
		Identity int64  `json:"identity"`
	}
	if err := json.NewDecoder(res.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("decode Cilium app identities: %w", err)
	}

	addresses := make(map[netip.Addr]struct{}, len(entries))
	for _, entry := range entries {
		prefix, err := netip.ParsePrefix(entry.CIDR)
		if err != nil {
			return nil, fmt.Errorf("parse Cilium app address: %w", err)
		}
		if entry.Identity > 0 && prefix.IsSingleIP() && prefix.Addr().IsPrivate() {
			addresses[prefix.Addr().Unmap()] = struct{}{}
		}
	}

	return addresses, nil
}
