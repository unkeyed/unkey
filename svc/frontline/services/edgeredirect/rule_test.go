package edgeredirect

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"testing"

	edgeredirectv1 "github.com/unkeyed/unkey/gen/proto/frontline/edgeredirect/v1"
)

func mkReq(t *testing.T, scheme, hostHeader, requestURI string) *http.Request {
	t.Helper()
	u, err := url.ParseRequestURI(requestURI)
	if err != nil {
		t.Fatalf("parse request URI %q: %v", requestURI, err)
	}
	req := &http.Request{
		Method: http.MethodGet,
		URL:    u,
		Host:   hostHeader,
	}
	if scheme == "https" {
		req.TLS = &tls.ConnectionState{}
	}
	return req
}

func TestRequireHTTPS(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		scheme  string
		host    string
		uri     string
		wantOK  bool
		wantLoc string
	}{
		{
			name:    "plain http with port is upgraded and port stripped",
			scheme:  "http",
			host:    "example.com:80",
			uri:     "/foo?x=1",
			wantOK:  true,
			wantLoc: "https://example.com/foo?x=1",
		},
		{
			name:    "plain http without port is upgraded",
			scheme:  "http",
			host:    "example.com",
			uri:     "/",
			wantOK:  true,
			wantLoc: "https://example.com/",
		},
		{
			name:    "ipv6 host preserves brackets",
			scheme:  "http",
			host:    "[::1]:80",
			uri:     "/health",
			wantOK:  true,
			wantLoc: "https://[::1]/health",
		},
		{
			name:    "raw query and fragment-free path are preserved",
			scheme:  "http",
			host:    "example.com",
			uri:     "/api/v2?token=abc&pretty=true",
			wantOK:  true,
			wantLoc: "https://example.com/api/v2?token=abc&pretty=true",
		},
		{
			name:    "empty path becomes /",
			scheme:  "http",
			host:    "example.com",
			uri:     "/",
			wantOK:  true,
			wantLoc: "https://example.com/",
		},
		{
			name:   "request already on https does not match",
			scheme: "https",
			host:   "example.com",
			uri:    "/",
			wantOK: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			loc, ok := applyRequireHTTPS(mkReq(t, tc.scheme, tc.host, tc.uri))
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if loc != tc.wantLoc {
				t.Fatalf("location = %q, want %q", loc, tc.wantLoc)
			}
		})
	}
}

func TestStripWWW(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		scheme  string
		host    string
		uri     string
		wantOK  bool
		wantLoc string
	}{
		{
			name:    "lowercase www is stripped",
			scheme:  "https",
			host:    "www.example.com",
			uri:     "/foo",
			wantOK:  true,
			wantLoc: "https://example.com/foo",
		},
		{
			name:    "case-insensitive prefix",
			scheme:  "https",
			host:    "WWW.Example.Com",
			uri:     "/",
			wantOK:  true,
			wantLoc: "https://Example.Com/",
		},
		{
			name:    "only first label stripped",
			scheme:  "https",
			host:    "www.www.example.com",
			uri:     "/",
			wantOK:  true,
			wantLoc: "https://www.example.com/",
		},
		{
			name:   "non-prefix www does not match",
			scheme: "https",
			host:   "nested.www.example.com",
			uri:    "/",
			wantOK: false,
		},
		{
			name:   "no www does not match",
			scheme: "https",
			host:   "example.com",
			uri:    "/",
			wantOK: false,
		},
		{
			name:    "preserves port",
			scheme:  "https",
			host:    "www.example.com:8443",
			uri:     "/",
			wantOK:  true,
			wantLoc: "https://example.com:8443/",
		},
		{
			name:    "preserves scheme on plain http",
			scheme:  "http",
			host:    "www.example.com",
			uri:     "/",
			wantOK:  true,
			wantLoc: "http://example.com/",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			loc, ok := applyStripWWW(mkReq(t, tc.scheme, tc.host, tc.uri))
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if loc != tc.wantLoc {
				t.Fatalf("location = %q, want %q", loc, tc.wantLoc)
			}
		})
	}
}

func TestAddWWW(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		scheme  string
		host    string
		uri     string
		wantOK  bool
		wantLoc string
	}{
		{
			name:    "apex gets www prepended",
			scheme:  "https",
			host:    "example.com",
			uri:     "/",
			wantOK:  true,
			wantLoc: "https://www.example.com/",
		},
		{
			name:   "already-www does not match",
			scheme: "https",
			host:   "www.example.com",
			uri:    "/",
			wantOK: false,
		},
		{
			name:   "case-insensitive www does not match",
			scheme: "https",
			host:   "WwW.Example.Com",
			uri:    "/",
			wantOK: false,
		},
		{
			name:   "single-label hosts skipped",
			scheme: "https",
			host:   "localhost",
			uri:    "/",
			wantOK: false,
		},
		{
			name:    "preserves port",
			scheme:  "https",
			host:    "example.com:8443",
			uri:     "/foo?x=1",
			wantOK:  true,
			wantLoc: "https://www.example.com:8443/foo?x=1",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			loc, ok := applyAddWWW(mkReq(t, tc.scheme, tc.host, tc.uri))
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if loc != tc.wantLoc {
				t.Fatalf("location = %q, want %q", loc, tc.wantLoc)
			}
		})
	}
}

func TestHostRewrite(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		scheme  string
		host    string
		uri     string
		from    string
		to      string
		wantOK  bool
		wantLoc string
	}{
		{
			name:    "exact match rewrites host",
			scheme:  "https",
			host:    "old.example.com",
			uri:     "/foo?x=1",
			from:    "old.example.com",
			to:      "new.example.com",
			wantOK:  true,
			wantLoc: "https://new.example.com/foo?x=1",
		},
		{
			name:    "case-insensitive match",
			scheme:  "https",
			host:    "OLD.example.com",
			uri:     "/",
			from:    "old.example.com",
			to:      "new.example.com",
			wantOK:  true,
			wantLoc: "https://new.example.com/",
		},
		{
			name:   "non-match falls through",
			scheme: "https",
			host:   "other.example.com",
			uri:    "/",
			from:   "old.example.com",
			to:     "new.example.com",
			wantOK: false,
		},
		{
			name:   "no suffix matching",
			scheme: "https",
			host:   "sub.old.example.com",
			uri:    "/",
			from:   "old.example.com",
			to:     "new.example.com",
			wantOK: false,
		},
		{
			name:    "preserves port",
			scheme:  "https",
			host:    "old.example.com:8443",
			uri:     "/",
			from:    "old.example.com",
			to:      "new.example.com",
			wantOK:  true,
			wantLoc: "https://new.example.com:8443/",
		},
		{
			name:   "empty from is no-op",
			scheme: "https",
			host:   "old.example.com",
			uri:    "/",
			from:   "",
			to:     "new.example.com",
			wantOK: false,
		},
		{
			name:   "empty to is no-op",
			scheme: "https",
			host:   "old.example.com",
			uri:    "/",
			from:   "old.example.com",
			to:     "",
			wantOK: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rule := &edgeredirectv1.HostRewrite{From: tc.from, To: tc.to}
			loc, ok := applyHostRewrite(mkReq(t, tc.scheme, tc.host, tc.uri), rule)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if loc != tc.wantLoc {
				t.Fatalf("location = %q, want %q", loc, tc.wantLoc)
			}
		})
	}
}

func TestStatusOrDefault(t *testing.T) {
	t.Parallel()
	if got := statusOrDefault(0); got != http.StatusPermanentRedirect {
		t.Fatalf("zero status -> %d, want %d", got, http.StatusPermanentRedirect)
	}
	if got := statusOrDefault(http.StatusMovedPermanently); got != http.StatusMovedPermanently {
		t.Fatalf("explicit 301 -> %d, want 301", got)
	}
}
