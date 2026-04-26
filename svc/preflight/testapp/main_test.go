package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestServer(t *testing.T, protocol string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(New(ListenerConfig{Protocol: protocol}))
	t.Cleanup(srv.Close)
	return srv
}

func TestMeta_EchoesInjectedEnv(t *testing.T) {
	t.Setenv("PORT", "9999")
	t.Setenv("UNKEY_DEPLOYMENT_ID", "dep_abc")
	t.Setenv("UNKEY_ENVIRONMENT_SLUG", "production")
	t.Setenv("UNKEY_REGION", "us-east-1")
	t.Setenv("UNKEY_INSTANCE_ID", "pod-xyz")
	t.Setenv("UNKEY_EPHEMERAL_DISK_PATH", "/tmp")

	srv := newTestServer(t, "h2c")

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/meta", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Preflight-Run-Id", "pflt_test")
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("X-Protocol"); got != "h2c" {
		t.Errorf("expected X-Protocol=h2c, got %q", got)
	}
	if got := resp.Header.Get("X-Preflight-Run-Id"); got != "pflt_test" {
		t.Errorf("expected X-Preflight-Run-Id=pflt_test, got %q", got)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	for k, want := range map[string]string{
		"unkey_deployment_id":       "dep_abc",
		"unkey_environment_slug":    "production",
		"unkey_region":              "us-east-1",
		"unkey_instance_id":         "pod-xyz",
		"unkey_ephemeral_disk_path": "/tmp",
		"protocol":                  "h2c",
	} {
		if body[k] != want {
			t.Errorf("body[%q] = %q, want %q", k, body[k], want)
		}
	}
}

func TestEnv_EchoesPreflightPrefixed(t *testing.T) {
	t.Setenv("PREFLIGHT_TOKEN", "uuid-value")
	srv := newTestServer(t, "http1")

	resp, err := srv.Client().Get(srv.URL + "/env?k=PREFLIGHT_TOKEN")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "uuid-value" {
		t.Fatalf("expected uuid-value, got %q", string(body))
	}
}

func TestEnv_RefusesNonPreflightKeys(t *testing.T) {
	t.Setenv("REAL_SECRET", "sensitive")
	srv := newTestServer(t, "http1")

	resp, err := srv.Client().Get(srv.URL + "/env?k=REAL_SECRET")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
}

func TestProbe_GetAndPost(t *testing.T) {
	srv := newTestServer(t, "http1")

	for _, method := range []string{http.MethodGet, http.MethodPost} {
		req, err := http.NewRequest(method, srv.URL+"/probe", nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := srv.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s /probe returned %d, want 200", method, resp.StatusCode)
		}
		_ = resp.Body.Close()
	}
}

func TestDisk_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("UNKEY_EPHEMERAL_DISK_PATH", dir)
	srv := newTestServer(t, "http1")

	payload := "hello-from-probe"

	putReq, _ := http.NewRequest(http.MethodPost, srv.URL+"/disk", bytes.NewBufferString(payload))
	putResp, err := srv.Client().Do(putReq)
	if err != nil {
		t.Fatal(err)
	}
	_ = putResp.Body.Close()
	if putResp.StatusCode != http.StatusNoContent {
		t.Fatalf("POST /disk: want 204, got %d", putResp.StatusCode)
	}

	getResp, err := srv.Client().Get(srv.URL + "/disk")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = getResp.Body.Close() }()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /disk: want 200, got %d", getResp.StatusCode)
	}
	got, _ := io.ReadAll(getResp.Body)
	if string(got) != payload {
		t.Fatalf("round-trip: want %q, got %q", payload, string(got))
	}

	if _, err := os.Stat(filepath.Join(dir, "preflight.txt")); err != nil {
		t.Fatalf("expected preflight.txt on disk, got error: %v", err)
	}
}

func TestDisk_MissingEnvReturns424(t *testing.T) {
	t.Setenv("UNKEY_EPHEMERAL_DISK_PATH", "")
	srv := newTestServer(t, "http1")

	resp, err := srv.Client().Get(srv.URL + "/disk")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusFailedDependency {
		t.Fatalf("expected 424, got %d", resp.StatusCode)
	}
}

func TestRegion_EchoesEnv(t *testing.T) {
	t.Setenv("UNKEY_REGION", "eu-central-1")
	srv := newTestServer(t, "http1")

	resp, err := srv.Client().Get(srv.URL + "/region")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	if strings.TrimSpace(string(body)) != "eu-central-1" {
		t.Fatalf("expected eu-central-1, got %q", string(body))
	}
}

func TestEmitMetric_RecordsLastEmission(t *testing.T) {
	srv := newTestServer(t, "http1")

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/emit-metric?name=foo&value=42", nil)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", resp.StatusCode)
	}

	lastMetricMu.Lock()
	defer lastMetricMu.Unlock()
	if lastMetric.Name != "foo" || lastMetric.Value != 42 {
		t.Fatalf("expected {foo, 42}, got %+v", lastMetric)
	}
}

func TestCPUSpin_RespectsDuration(t *testing.T) {
	srv := newTestServer(t, "http1")

	resp, err := srv.Client().Get(srv.URL + "/cpu-spin?ms=20")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestCPUSpin_RejectsInvalidDuration(t *testing.T) {
	srv := newTestServer(t, "http1")

	resp, err := srv.Client().Get(srv.URL + "/cpu-spin?ms=notanumber")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
