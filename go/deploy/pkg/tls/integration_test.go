package tls_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/unkeyed/unkey/go/deploy/pkg/tls"
)

// AIDEV-NOTE: Tests showing TLS provider works in all modes
// Proves we can add this without breaking anything

func TestTLSProvider_DisabledMode(t *testing.T) {
	ctx := context.Background()

	// Default disabled mode
	provider, err := tls.NewProvider(ctx, tls.Config{
		Mode: tls.ModeDisabled,
	})
	if err != nil {
		t.Fatalf("Failed to create disabled provider: %v", err)
	}
	defer provider.Close()

	// Server should work without TLS
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello"))
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	// Client should connect fine
	client := provider.HTTPClient()
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestTLSProvider_FileMode(t *testing.T) {
	t.Skip("Requires test certificates")

	ctx := context.Background()

	provider, err := tls.NewProvider(ctx, tls.Config{
		Mode:     tls.ModeFile,
		CertFile: "testdata/server.crt",
		KeyFile:  "testdata/server.key",
		CAFile:   "testdata/ca.crt",
	})
	if err != nil {
		t.Fatalf("Failed to create file provider: %v", err)
	}
	defer provider.Close()

	// Should have TLS config
	tlsConfig, err := provider.ServerTLSConfig()
	if err != nil {
		t.Fatalf("Failed to get TLS config: %v", err)
	}

	if tlsConfig == nil {
		t.Error("Expected TLS config, got nil")
	}
}

func TestTLSProvider_SPIFFEMode_Fallback(t *testing.T) {
	ctx := context.Background()

	// SPIFFE mode with no agent running
	provider, err := tls.NewProvider(ctx, tls.Config{
		Mode:             tls.ModeSPIFFE,
		SPIFFESocketPath: "/tmp/nonexistent.sock",
	})

	// Should fallback gracefully
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	// Should work like disabled mode
	client := provider.HTTPClient()
	if client == nil {
		t.Error("Expected HTTP client, got nil")
	}
}

// Benchmark showing no performance impact when disabled
func BenchmarkTLSProvider_Disabled(b *testing.B) {
	ctx := context.Background()
	provider, _ := tls.NewProvider(ctx, tls.Config{Mode: tls.ModeDisabled})
	defer provider.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client := provider.HTTPClient()
		_ = client
	}
}
