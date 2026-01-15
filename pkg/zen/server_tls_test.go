package zen

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	tlspkg "github.com/unkeyed/unkey/pkg/tls"
)

// generateTestCertificate creates a self-signed certificate and key for testing.
func generateTestCertificate(t *testing.T) (certPEM []byte, keyPEM []byte) {
	t.Helper()

	// Generate a private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "Failed to generate private key")

	// Create a template for the certificate
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	require.NoError(t, err, "Failed to generate serial number")

	notBefore := time.Now()
	notAfter := notBefore.Add(time.Hour) // Valid for an hour

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Unkey Test"},
			CommonName:   "localhost",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
	}

	// Create the certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	require.NoError(t, err, "Failed to create certificate")

	// Encode the certificate to PEM
	certBuffer := &bytes.Buffer{}
	err = pem.Encode(certBuffer, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	require.NoError(t, err, "Failed to encode certificate to PEM")

	// Encode the private key to PEM
	keyBuffer := &bytes.Buffer{}
	err = pem.Encode(keyBuffer, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	require.NoError(t, err, "Failed to encode private key to PEM")

	return certBuffer.Bytes(), keyBuffer.Bytes()
}

// TestServerWithTLS tests that the server can be configured with TLS.
func TestServerWithTLS(t *testing.T) {
	// Generate test certificate and key
	certPEM, keyPEM := generateTestCertificate(t)

	// Create a TLS config
	tlsConfig, err := tlspkg.New(certPEM, keyPEM)
	require.NoError(t, err, "Failed to create TLS config")

	// Create a mock logger
	logger := logging.New()

	// Create a server with TLS config
	server, err := New(Config{
		Logger: logger,
		TLS:    tlsConfig,
	})
	require.NoError(t, err, "Failed to create server with TLS config")

	// Register a test route
	testRoute := NewRoute("GET", "/test", func(ctx context.Context, s *Session) error {
		return s.JSON(http.StatusOK, map[string]string{
			"status": "ok",
		})
	})
	server.RegisterRoute([]Middleware{}, testRoute)

	// Create ephemeral listener
	ln, err := net.Listen("tcp", ":0")
	require.NoError(t, err, "Failed to create ephemeral listener")

	// Get the address for the test client
	addr := ln.Addr().String()

	// Start the server in a goroutine
	serverCtx, serverCancel := context.WithCancel(context.Background())
	defer serverCancel()

	// Create a channel to signal when the server is ready
	serverReady := make(chan struct{})

	go func() {
		// Signal that we're about to start the server
		close(serverReady)

		listenErr := server.Serve(serverCtx, ln)
		if listenErr != nil && listenErr.Error() != "http: Server closed" {
			t.Errorf("server.Serve returned: %v", listenErr)
		}
	}()
	defer server.Shutdown(context.Background())

	// Wait for the server to signal it's starting
	<-serverReady

	// Create a custom HTTP client that skips certificate verification (since we're using a self-signed cert)
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	// Wait for server to be ready to accept connections
	require.Eventually(t, func() bool {
		resp, err := client.Get("https://" + addr + "/test")
		if err != nil {
			return false
		}
		resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 2*time.Second, 10*time.Millisecond, "server should be ready")

	// Make a request to the server
	resp, err := client.Get("https://" + addr + "/test")
	require.NoError(t, err, "Failed to make request to HTTPS server")
	defer resp.Body.Close()

	// Verify the response
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status code 200")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body")
	assert.Contains(t, string(body), "ok", "Response body should contain 'ok'")
}

// TestServerWithInvalidTLS tests error handling for invalid TLS configurations
func TestServerWithTLSCertificateVerificationFailure(t *testing.T) {
	// Generate test certificate and key
	certPEM, keyPEM := generateTestCertificate(t)

	// Corrupt the certificate to make it invalid
	corruptedCertPEM := append([]byte("CORRUPTED"), certPEM...)

	// Try to create a TLS config - this should fail
	_, err := tlspkg.New(corruptedCertPEM, keyPEM)
	require.Error(t, err, "Expected TLS config creation to fail with corrupted cert")
	assert.Contains(t, err.Error(), "failed to parse", "Error should mention parsing failure")
}

// TestServerWithTLSContextCancellation tests that the server properly shuts down
// when the context is canceled
func TestServerWithTLSContextCancellation(t *testing.T) {
	// Generate test certificate and key
	certPEM, keyPEM := generateTestCertificate(t)

	// Create a TLS config
	tlsConfig, err := tlspkg.New(certPEM, keyPEM)
	require.NoError(t, err, "Failed to create TLS config")

	// Create a mock logger
	logger := logging.New()

	// Create a server with TLS config
	server, err := New(Config{
		Logger: logger,
		TLS:    tlsConfig,
	})
	require.NoError(t, err, "Failed to create server with TLS config")

	// Register a test route
	testRoute := NewRoute("GET", "/test", func(ctx context.Context, s *Session) error {
		return s.JSON(http.StatusOK, map[string]string{
			"status": "ok",
		})
	})
	server.RegisterRoute([]Middleware{}, testRoute)

	// Create ephemeral listener
	ln, err := net.Listen("tcp", ":0")
	require.NoError(t, err, "Failed to create ephemeral listener")

	// Get the address for the test client
	addr := ln.Addr().String()

	// Create a context that can be canceled
	serverCtx, serverCancel := context.WithCancel(context.Background())

	// Create a channel to signal when the server is ready
	serverReady := make(chan struct{})

	// Create a channel to signal when the server is done
	serverDone := make(chan struct{})

	// Start the server in a goroutine
	go func() {
		// Signal that we're about to start the server
		close(serverReady)

		listenErr := server.Serve(serverCtx, ln)
		if listenErr != nil && listenErr.Error() != "http: Server closed" {
			t.Errorf("server.Serve returned: %v", listenErr)
		}

		// Signal that the server has exited
		close(serverDone)
	}()

	// Wait for the server to signal it's starting
	<-serverReady

	// Create a custom HTTP client that skips certificate verification
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	// Wait for server to be ready to accept connections
	require.Eventually(t, func() bool {
		resp, err := client.Get("https://" + addr + "/test")
		if err != nil {
			return false
		}
		resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 2*time.Second, 10*time.Millisecond, "server should be ready")

	// Make a request to verify the server is running
	resp, err := client.Get("https://" + addr + "/test")
	require.NoError(t, err, "Failed to make request to HTTPS server")
	resp.Body.Close()

	// Cancel the context to trigger server shutdown
	serverCancel()

	// Wait for the server to exit with a timeout
	select {
	case <-serverDone:
		// Server exited as expected
	case <-time.After(5 * time.Second):
		t.Fatal("Server did not shut down within timeout after context cancellation")
	}

	// Verify the server is no longer accepting connections
	_, err = client.Get("https://" + addr + "/test")
	require.Error(t, err, "Expected connection error after server shutdown")
}
