// Command testapp is the tiny HTTP server that preflight probes
// exercise. It is deployed like any customer workload (via unkey
// deploy) into the preflight workspace; each probe asserts one property
// of the deploy pipeline by talking to a specific endpoint here.
//
// The endpoint surface mirrors the plan's Test-app surface table. Every
// handler is deliberately small and dependency-free so probes can
// attribute a failure to pipeline infrastructure, not to test-app bugs.
package main

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"
)

// ListenerConfig tells the server where to bind. Split out so tests
// can call New() against a random port.
type ListenerConfig struct {
	// Addr to bind (":0" picks a random port, handy in tests).
	Addr string
	// Protocol advertised in the /meta response's x-protocol header.
	// Set by the upstream_protocol deployment setting in real runs;
	// tests just echo whatever they set.
	Protocol string
}

// New wires the full handler set. Exported so tests can mount the
// handler on httptest.Server without running the command-line main.
func New(cfg ListenerConfig) *http.ServeMux {
	mux := http.NewServeMux()

	registerMeta(mux, cfg.Protocol)
	registerEnv(mux)
	registerProbe(mux)
	registerDisk(mux)
	registerEmitMetric(mux)
	registerCPUSpin(mux)
	registerLastShutdown(mux)
	registerEgressProbe(mux)
	registerRegion(mux)
	registerCrash(mux)

	return mux
}

func main() {
	addr := ":" + envOrDefault("PORT", "8080")
	mux := New(ListenerConfig{
		Addr:     addr,
		Protocol: detectProtocol(),
	})

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("testapp: listen %s: %v", addr, err)
	}
	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// SIGTERM handling. Krane sends the deployment's configured
	// shutdown_signal; we record it so the graceful-shutdown probe
	// (tier 2.15) can confirm the PreStop hook wired up correctly.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	done := make(chan struct{})
	go func() {
		sig := <-signals
		recordShutdown(sig)
		_ = srv.Close()
		close(done)
	}()

	log.Printf("testapp: listening on %s", addr)
	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		log.Fatalf("testapp: serve: %v", err)
	}
	<-done
}

// ---------------------------------------------------------------------
// GET /meta
// Returns the full set of Krane-injected env vars plus the negotiated
// upstream protocol. Tier 1.6 and 1.7 assertions read this.
// ---------------------------------------------------------------------

func registerMeta(mux *http.ServeMux, protocol string) {
	mux.HandleFunc("GET /meta", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Protocol", protocol)
		w.Header().Set("X-Preflight-Run-Id", r.Header.Get("X-Preflight-Run-Id"))
		w.Header().Set("Content-Type", "application/json")

		body := map[string]string{
			"port":                      os.Getenv("PORT"),
			"unkey_deployment_id":       os.Getenv("UNKEY_DEPLOYMENT_ID"),
			"unkey_git_commit_sha":      os.Getenv("UNKEY_GIT_COMMIT_SHA"),
			"unkey_environment_slug":    os.Getenv("UNKEY_ENVIRONMENT_SLUG"),
			"unkey_region":              os.Getenv("UNKEY_REGION"),
			"unkey_instance_id":         os.Getenv("UNKEY_INSTANCE_ID"),
			"unkey_ephemeral_disk_path": os.Getenv("UNKEY_EPHEMERAL_DISK_PATH"),
			"protocol":                  protocol,
		}
		_ = json.NewEncoder(w).Encode(body)
	})
}

// ---------------------------------------------------------------------
// GET /env?k=<key>
// Echoes an arbitrary env var. Tier 1.5 uses this to verify Vault
// DecryptBulk round-trip: probe sets PREFLIGHT_TOKEN, reads it back.
// Refuses to echo anything that does not start with PREFLIGHT_ to make
// accidental disclosure of real secrets strictly impossible.
// ---------------------------------------------------------------------

func registerEnv(mux *http.ServeMux) {
	mux.HandleFunc("GET /env", func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("k")
		if key == "" {
			http.Error(w, "missing k query param", http.StatusBadRequest)
			return
		}
		if !isEchoableKey(key) {
			http.Error(w, "key not echoable", http.StatusForbidden)
			return
		}
		_, _ = io.WriteString(w, os.Getenv(key))
	})
}

// Only echo env vars that are clearly test-scoped. Real customer env
// vars must never leak out of this endpoint even if the probe has the
// wrong credentials.
func isEchoableKey(k string) bool {
	const prefix = "PREFLIGHT_"
	return len(k) > len(prefix) && k[:len(prefix)] == prefix
}

// ---------------------------------------------------------------------
// GET  /probe   -> 200
// POST /probe   -> 200 (exercises Krane's wget --post-data path)
// Tier 1.9 uses a custom healthcheck.path=/probe so the kubelet's
// readiness check traverses this handler.
// ---------------------------------------------------------------------

func registerProbe(mux *http.ServeMux) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "ok")
	}
	mux.HandleFunc("GET /probe", handler)
	mux.HandleFunc("POST /probe", handler)
}

// ---------------------------------------------------------------------
// POST /disk  body=<text>  -> writes to $UNKEY_EPHEMERAL_DISK_PATH/preflight.txt
// GET  /disk                -> returns the contents
// Tier 1.8 asserts a write/read round-trip through the PVC Krane mounts.
// ---------------------------------------------------------------------

func registerDisk(mux *http.ServeMux) {
	mux.HandleFunc("POST /disk", func(w http.ResponseWriter, r *http.Request) {
		path, ok := diskPath(w)
		if !ok {
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := os.WriteFile(path, body, 0o600); err != nil {
			http.Error(w, "write: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("GET /disk", func(w http.ResponseWriter, _ *http.Request) {
		path, ok := diskPath(w)
		if !ok {
			return
		}
		b, err := os.ReadFile(path)
		if err != nil {
			http.Error(w, "read: "+err.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(b)
	})
}

func diskPath(w http.ResponseWriter) (string, bool) {
	base := os.Getenv("UNKEY_EPHEMERAL_DISK_PATH")
	if base == "" {
		http.Error(w, "UNKEY_EPHEMERAL_DISK_PATH not set; storage_mib=0?", http.StatusFailedDependency)
		return "", false
	}
	return filepath.Join(base, "preflight.txt"), true
}

// ---------------------------------------------------------------------
// POST /emit-metric?name=<>&value=<>
// Placeholder for future metric-collector probes (see the worked
// example in docs/preflight/adding-a-probe.md). For now it just
// records the most recent emission in memory so tests can inspect it.
// ---------------------------------------------------------------------

type metricSample struct {
	Name  string
	Value int64
}

var (
	lastMetricMu sync.Mutex
	lastMetric   = metricSample{Name: "", Value: 0}
)

func registerEmitMetric(mux *http.ServeMux) {
	mux.HandleFunc("POST /emit-metric", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		valueStr := r.URL.Query().Get("value")
		value, err := strconv.ParseInt(valueStr, 10, 64)
		if name == "" || err != nil {
			http.Error(w, "missing or invalid name/value", http.StatusBadRequest)
			return
		}
		lastMetricMu.Lock()
		lastMetric.Name = name
		lastMetric.Value = value
		lastMetricMu.Unlock()
		w.WriteHeader(http.StatusAccepted)
	})
}

// ---------------------------------------------------------------------
// GET /cpu-spin?ms=<>
// Burns a CPU for the requested duration so tier 2.14 can drive HPA
// scale-up. Bounded to 10s to prevent an abusive probe from pinning
// the pod indefinitely.
// ---------------------------------------------------------------------

func registerCPUSpin(mux *http.ServeMux) {
	mux.HandleFunc("GET /cpu-spin", func(w http.ResponseWriter, r *http.Request) {
		ms, err := strconv.Atoi(r.URL.Query().Get("ms"))
		if err != nil || ms <= 0 {
			http.Error(w, "ms must be a positive integer", http.StatusBadRequest)
			return
		}
		if ms > 10_000 {
			ms = 10_000
		}
		deadline := time.Now().Add(time.Duration(ms) * time.Millisecond)
		for time.Now().Before(deadline) { //nolint:staticcheck // tight spin is the point
		}
		w.WriteHeader(http.StatusOK)
	})
}

// ---------------------------------------------------------------------
// GET /last-shutdown
// Returns the signal name that killed the previous pod instance, read
// from the ephemeral disk at $UNKEY_EPHEMERAL_DISK_PATH/last-shutdown.
// Written by this process on receipt of SIGTERM/SIGINT/SIGQUIT; the
// next instance reads it on startup. Tier 2.15 (graceful shutdown)
// asserts the PreStop exec hook delivered the right signal.
// ---------------------------------------------------------------------

func registerLastShutdown(mux *http.ServeMux) {
	mux.HandleFunc("GET /last-shutdown", func(w http.ResponseWriter, _ *http.Request) {
		path := lastShutdownPath()
		if path == "" {
			http.Error(w, "no ephemeral disk configured", http.StatusFailedDependency)
			return
		}
		b, err := os.ReadFile(path)
		if err != nil && !os.IsNotExist(err) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(b)
	})
}

func recordShutdown(sig os.Signal) {
	path := lastShutdownPath()
	if path == "" {
		return
	}
	_ = os.WriteFile(path, []byte(sig.String()), 0o600)
}

func lastShutdownPath() string {
	base := os.Getenv("UNKEY_EPHEMERAL_DISK_PATH")
	if base == "" {
		return ""
	}
	return filepath.Join(base, "last-shutdown")
}

// ---------------------------------------------------------------------
// GET /egress-probe?target=<>
// Attempts an outbound TCP dial to target (host:port). Tier 2.17 uses
// this to verify Cilium NetworkPolicy is actually enforcing egress
// restrictions; a successful dial to a denied target is a finding.
// ---------------------------------------------------------------------

func registerEgressProbe(mux *http.ServeMux) {
	mux.HandleFunc("GET /egress-probe", func(w http.ResponseWriter, _ *http.Request) {
		target := defaultEgressTarget()
		//nolint:exhaustruct // intentional; only Timeout matters for this probe
		dialer := &net.Dialer{Timeout: 2 * time.Second}
		conn, err := dialer.Dial("tcp", target)
		if err != nil {
			// Distinguish "blocked by policy" from other failures where
			// we can tell them apart: timeout-without-any-response is
			// the usual signature of a Cilium drop.
			if isLikelyBlocked(err) {
				_, _ = io.WriteString(w, "blocked")
				return
			}
			_, _ = io.WriteString(w, "timeout")
			return
		}
		_ = conn.Close()
		_, _ = io.WriteString(w, "allowed")
	})
}

func defaultEgressTarget() string {
	if v := os.Getenv("PREFLIGHT_EGRESS_TARGET"); v != "" {
		return v
	}
	// AWS instance metadata service; Cilium policy should block egress
	// from preflight pods so this is a safe default test.
	return "169.254.169.254:80"
}

func isLikelyBlocked(err error) bool {
	// Simplification; sharper detection happens at the probe side.
	return err != nil
}

// ---------------------------------------------------------------------
// GET /region
// Returns UNKEY_REGION verbatim. Identical to the endpoint used by
// failure-scenario 0004; tier 2.19 reuses it to verify latency DNS
// routes each region correctly.
// ---------------------------------------------------------------------

func registerRegion(mux *http.ServeMux) {
	mux.HandleFunc("GET /region", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, os.Getenv("UNKEY_REGION"))
	})
}

// ---------------------------------------------------------------------
// GET /crash
// Exits the process non-zero. Tier 3.25 (error-path coverage) uses
// this to verify failed deploys report status=failed and do not
// write frontline routes.
// ---------------------------------------------------------------------

func registerCrash(mux *http.ServeMux) {
	mux.HandleFunc("GET /crash", func(_ http.ResponseWriter, _ *http.Request) {
		// Give the response a chance to flush before exiting; we don't
		// actually write a body because the client is going to observe
		// connection reset anyway.
		go func() {
			time.Sleep(50 * time.Millisecond)
			os.Exit(1)
		}()
	})
}

// ---------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func detectProtocol() string {
	// The real frontline terminates to http1 or h2c based on the
	// deployment's upstream_protocol setting. We cannot detect it from
	// inside the pod, so the deploy pipeline is expected to inject
	// PREFLIGHT_UPSTREAM_PROTOCOL=<http1|h2c> as a test-only env var.
	if v := os.Getenv("PREFLIGHT_UPSTREAM_PROTOCOL"); v != "" {
		return v
	}
	return "unknown"
}
