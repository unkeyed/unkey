package httpx_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/unkeyed/unkey/pkg/httpx"
)

type echoReq struct {
	Hello string `json:"hello"`
}

type echoResp struct {
	Got    string `json:"got"`
	Method string `json:"method"`
}

func TestGet_DecodesJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(echoResp{Got: "ok", Method: r.Method})
	}))
	t.Cleanup(srv.Close)

	got, err := httpx.Get[echoResp](context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got.Method != http.MethodGet || got.Got != "ok" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestPost_RoundTripsJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q; want application/json", ct)
		}
		var in echoReq
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			t.Errorf("decode req: %v", err)
		}
		_ = json.NewEncoder(w).Encode(echoResp{Got: in.Hello, Method: r.Method})
	}))
	t.Cleanup(srv.Close)

	got, err := httpx.Post[echoReq, echoResp](context.Background(), srv.URL, echoReq{Hello: "world"})
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	if got.Got != "world" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestStatusError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, "no such resource")
	}))
	t.Cleanup(srv.Close)

	_, err := httpx.Get[echoResp](context.Background(), srv.URL)
	if !httpx.IsStatus(err, http.StatusNotFound) {
		t.Fatalf("want 404 StatusError, got %v", err)
	}
	if !strings.Contains(err.Error(), "no such resource") {
		t.Fatalf("body not in error message: %v", err)
	}
}

func TestHeader_OverridesAndAdds(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer xyz" {
			t.Errorf("Authorization = %q; want Bearer xyz", got)
		}
		if got := r.Header.Get("X-Custom"); got != "v1" {
			t.Errorf("X-Custom = %q; want v1", got)
		}
		_ = json.NewEncoder(w).Encode(echoResp{Got: "ok"})
	}))
	t.Cleanup(srv.Close)

	_, err := httpx.Get[echoResp](context.Background(), srv.URL,
		httpx.Bearer("xyz"),
		httpx.Header("X-Custom", "v1"),
	)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
}

func TestEmpty_DiscardsBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A response that is NOT valid JSON. Empty must skip the
		// decode step entirely, so this should not error out.
		_, _ = io.WriteString(w, "ok plaintext")
	}))
	t.Cleanup(srv.Close)

	if _, err := httpx.Get[httpx.Empty](context.Background(), srv.URL); err != nil {
		t.Fatalf("Get[Empty]: %v", err)
	}
}

func TestEmptyRequest_NoBodyOnWire(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if len(body) != 0 {
			t.Errorf("request body = %q; want empty", string(body))
		}
		if ct := r.Header.Get("Content-Type"); ct != "" {
			t.Errorf("Content-Type set to %q on bodyless request", ct)
		}
		_ = json.NewEncoder(w).Encode(echoResp{Got: "ok"})
	}))
	t.Cleanup(srv.Close)

	if _, err := httpx.Post[httpx.Empty, echoResp](context.Background(), srv.URL, httpx.Empty{}); err != nil {
		t.Fatalf("Post[Empty,_]: %v", err)
	}
}

type recordingDoer struct {
	hits  int
	inner http.RoundTripper
}

func (r *recordingDoer) Do(req *http.Request) (*http.Response, error) {
	r.hits++
	if r.inner == nil {
		r.inner = http.DefaultTransport
	}
	return r.inner.RoundTrip(req)
}

func TestWithDoer_OverridesDefaultClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(echoResp{Got: "ok"})
	}))
	t.Cleanup(srv.Close)

	d := &recordingDoer{}
	if _, err := httpx.Get[echoResp](context.Background(), srv.URL, httpx.WithDoer(d)); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if d.hits != 1 {
		t.Fatalf("recordingDoer hits = %d; want 1", d.hits)
	}
}
