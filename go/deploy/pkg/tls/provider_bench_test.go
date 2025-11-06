package tls

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// BenchmarkGetCertificate measures the overhead of loading certificates on each connection
func BenchmarkGetCertificate(b *testing.B) {
	// Create temporary directory for test certificates
	tmpDir, err := os.MkdirTemp("", "tls-bench-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate test certificate
	certFile := filepath.Join(tmpDir, "cert.pem")
	keyFile := filepath.Join(tmpDir, "key.pem")
	if err = generateTestCert(certFile, keyFile); err != nil {
		b.Fatal(err)
	}

	// Create provider
	provider, err := NewProvider(context.Background(), Config{
		Mode:     ModeFile,
		CertFile: certFile,
		KeyFile:  keyFile,
	})
	if err != nil {
		b.Fatal(err)
	}
	defer provider.Close()

	// Get TLS config
	tlsConfig, err := provider.ServerTLSConfig()
	if err != nil {
		b.Fatal(err)
	}

	b.Run("GetCertificate", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate TLS handshake calling GetCertificate
			cert, err := tlsConfig.GetCertificate(&tls.ClientHelloInfo{
				ServerName: "test.example.com",
			})
			if err != nil {
				b.Fatal(err)
			}
			if cert == nil {
				b.Fatal("expected certificate")
			}
		}
	})

	// Benchmark with cached approach for comparison
	b.Run("GetCertificateCached", func(b *testing.B) {
		// Load cert once
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			b.Fatal(err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Return cached cert
			_ = &cert
		}
	})
}

// BenchmarkTLSHandshake measures full TLS handshake overhead
func BenchmarkTLSHandshake(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "tls-bench-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	certFile := filepath.Join(tmpDir, "cert.pem")
	keyFile := filepath.Join(tmpDir, "key.pem")
	if err := generateTestCert(certFile, keyFile); err != nil {
		b.Fatal(err)
	}

	// Test both approaches
	benchmarks := []struct {
		name      string
		tlsConfig func() (*tls.Config, error)
	}{
		{
			name: "DynamicLoad",
			tlsConfig: func() (*tls.Config, error) {
				provider, err := NewProvider(context.Background(), Config{
					Mode:     ModeFile,
					CertFile: certFile,
					KeyFile:  keyFile,
				})
				if err != nil {
					return nil, err
				}
				return provider.ServerTLSConfig()
			},
		},
		{
			name: "StaticLoad",
			tlsConfig: func() (*tls.Config, error) {
				cert, err := tls.LoadX509KeyPair(certFile, keyFile)
				if err != nil {
					return nil, err
				}
				return &tls.Config{
					Certificates: []tls.Certificate{cert},
					MinVersion:   tls.VersionTLS13,
				}, nil
			},
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			tlsConfig, err := bm.tlsConfig()
			if err != nil {
				b.Fatal(err)
			}

			// Create test server
			ln, err := tls.Listen("tcp", "127.0.0.1:0", tlsConfig)
			if err != nil {
				b.Fatal(err)
			}
			defer ln.Close()

			// Accept connections in background
			done := make(chan struct{})
			go func() {
				for {
					select {
					case <-done:
						return
					default:
						conn, err := ln.Accept()
						if err != nil {
							return
						}
						// Perform TLS handshake
						go func(c net.Conn) {
							defer c.Close()
							if tlsConn, ok := c.(*tls.Conn); ok {
								_ = tlsConn.Handshake()
							}
						}(conn)
					}
				}
			}()
			defer close(done)

			// Benchmark client connections
			addr := ln.Addr().String()
			clientConfig := &tls.Config{
				InsecureSkipVerify: true,
			}

			// Small delay to ensure server is ready
			time.Sleep(10 * time.Millisecond)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				conn, err := tls.Dial("tcp", addr, clientConfig)
				if err != nil {
					b.Fatal(err)
				}
				conn.Close()
			}
		})
	}
}

// generateTestCert creates a self-signed certificate for testing
func generateTestCert(certFile, keyFile string) error {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1)},
		DNSNames:              []string{"localhost", "test.example.com"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return err
	}

	// Write cert
	certOut, err := os.Create(certFile)
	if err != nil {
		return err
	}
	defer certOut.Close()

	if err = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return err
	}

	// Write key
	keyOut, err := os.Create(keyFile)
	if err != nil {
		return err
	}
	defer keyOut.Close()

	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return err
	}
	return pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privDER})
}

// BenchmarkConcurrentGetCertificate measures performance under concurrent load
func BenchmarkConcurrentGetCertificate(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "tls-bench-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	certFile := filepath.Join(tmpDir, "cert.pem")
	keyFile := filepath.Join(tmpDir, "key.pem")
	if err := generateTestCert(certFile, keyFile); err != nil {
		b.Fatal(err)
	}

	// Test both uncached and cached versions
	benchmarks := []struct {
		name   string
		config Config
	}{
		{
			name: "Uncached",
			config: Config{
				Mode:     ModeFile,
				CertFile: certFile,
				KeyFile:  keyFile,
			},
		},
		{
			name: "Cached-5s",
			config: Config{
				Mode:              ModeFile,
				CertFile:          certFile,
				KeyFile:           keyFile,
				EnableCertCaching: true,
				CertCacheTTL:      5 * time.Second,
			},
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			provider, err := NewProvider(context.Background(), bm.config)
			if err != nil {
				b.Fatal(err)
			}
			defer provider.Close()

			tlsConfig, err := provider.ServerTLSConfig()
			if err != nil {
				b.Fatal(err)
			}

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					cert, err := tlsConfig.GetCertificate(&tls.ClientHelloInfo{
						ServerName: "test.example.com",
					})
					if err != nil {
						b.Fatal(err)
					}
					if cert == nil {
						b.Fatal("expected certificate")
					}
				}
			})
		})
	}
}
