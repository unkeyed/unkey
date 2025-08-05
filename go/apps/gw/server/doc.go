// Package server provides a lightweight HTTP gateway server with multi-domain TLS termination
// and reverse proxy capabilities.
//
// The server is designed to act as an API gateway, handling incoming requests
// and forwarding them to backend services. It includes:
//
//   - Multi-domain TLS termination with SNI support
//   - Dynamic certificate management
//   - Configurable middleware (logging, metrics, tracing, panic recovery)
//   - Flexible reverse proxy implementation
//   - Round-robin load balancing
//   - Host-based routing
//
// Example usage:
//
//	// Create certificate manager for multiple domains
//	certManager := server.NewSimpleCertManager(logger)
//	certManager.LoadCertificate("example.com", "/certs/example.com/cert.pem", "/certs/example.com/key.pem")
//	certManager.LoadCertificate("test123.com", "/certs/test123.com/cert.pem", "/certs/test123.com/key.pem")
//
//	// Create host-based routing
//	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	    var backend string
//	    switch r.Host {
//	    case "example.com":
//	        backend = "http://backend-example:8080"
//	    case "test123.com":
//	        backend = "http://backend-test123:8080"
//	    default:
//	        backend = "http://default-backend:8080"
//	    }
//
//	    proxy, _ := server.SimpleProxy(backend, logger)
//	    proxy.ServeHTTP(w, r)
//	})
//
//	// Apply middleware
//	finalHandler := server.Chain(
//	    server.WithPanicRecovery(logger),
//	    server.WithTracing(),
//	    server.WithMetrics(),
//	    server.WithLogging(logger),
//	)(handler)
//
//	// Create server with TLS
//	srv, err := server.New(server.Config{
//	    Logger:      logger,
//	    Handler:     finalHandler,
//	    CertManager: certManager,
//	    EnableTLS:   true,
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Start listening with SNI-based TLS termination
//	if err := srv.Serve(ctx, listener); err != nil {
//	    log.Fatal(err)
//	}
package server
