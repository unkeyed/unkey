package collector

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/unkeyed/unkey/pkg/retry"
	"github.com/unkeyed/unkey/svc/heimdall/pkg/metrics"
)

// kubelet summary API response types — only fields we need.

type kubeletSummary struct {
	Pods []kubeletPod `json:"pods"`
}

type kubeletPod struct {
	PodRef     kubeletPodRef      `json:"podRef"`
	Containers []kubeletContainer `json:"containers"`
	Network    *kubeletPodNetwork `json:"network"`
}

type kubeletPodRef struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Labels    map[string]string `json:"labels"`
}

type kubeletContainer struct {
	Name   string        `json:"name"`
	CPU    kubeletCPU    `json:"cpu"`
	Memory kubeletMemory `json:"memory"`
}

type kubeletCPU struct {
	UsageCoreNanoSeconds *uint64 `json:"usageCoreNanoSeconds"`
}

type kubeletMemory struct {
	WorkingSetBytes *uint64 `json:"workingSetBytes"`
}

type kubeletPodNetwork struct {
	Interfaces []kubeletNetworkInterface `json:"interfaces"`
}

type kubeletNetworkInterface struct {
	Name    string `json:"name"`
	TxBytes uint64 `json:"txBytes"`
	RxBytes uint64 `json:"rxBytes"`
}

var kubeletRetry = retry.New(
	retry.Attempts(3),
	retry.Backoff(func(n int) time.Duration {
		return time.Duration(n) * 500 * time.Millisecond
	}),
)

// fetchSummary queries the kubelet stats summary API with retry.
func (c *Collector) fetchSummary(ctx context.Context) (*kubeletSummary, error) {
	return retry.DoWithResultContext(kubeletRetry, ctx, func() (*kubeletSummary, error) {
		summary, err := c.doFetchSummary(ctx)
		if err != nil {
			metrics.KubeletFetchErrors.Inc()
		}
		return summary, err
	})
}

func (c *Collector) doFetchSummary(ctx context.Context) (*kubeletSummary, error) {
	token, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return nil, fmt.Errorf("reading service account token: %w", err)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // kubelet uses self-signed certs
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.kubeletStatsURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+string(token))

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("kubelet returned %d: %s", resp.StatusCode, string(body))
	}

	var summary kubeletSummary
	if err := json.NewDecoder(resp.Body).Decode(&summary); err != nil {
		return nil, fmt.Errorf("decoding kubelet summary: %w", err)
	}

	return &summary, nil
}
