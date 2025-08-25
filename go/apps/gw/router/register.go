package router

import (
	"net/http"
	"strings"
	"time"

	"github.com/unkeyed/unkey/go/apps/gw/router/acme_challenge"
	"github.com/unkeyed/unkey/go/apps/gw/router/gateway_proxy"
	"github.com/unkeyed/unkey/go/apps/gw/server"
	"github.com/unkeyed/unkey/go/apps/gw/services/auth"
	"github.com/unkeyed/unkey/go/apps/gw/services/proxy"
)

// ServerType indicates which type of server to configure
type ServerType string

const (
	HTTPServer  ServerType = "http"
	HTTPSServer ServerType = "https"
)

// Register registers all routes with the gateway server.
// This function runs during startup and sets up the middleware chain and routes.
// The serverType parameter determines whether to register HTTP (ACME) or HTTPS (main gateway) routes.
func Register(srv *server.Server, svc *Services, region string, serverType ServerType) {
	// Define default middleware stack
	defaultMiddlewares := []server.Middleware{
		server.WithTracing(),
		server.WithMetrics(svc.ClickHouse, region),
		server.WithLogging(svc.Logger),
		server.WithPanicRecovery(svc.Logger),
		server.WithErrorHandling(svc.Logger),
	}

	// Create shared transport for connection pooling and HTTP/2 support
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		// Enable HTTP/2
		ForceAttemptHTTP2: true,
	}

	// Create proxy service with shared transport
	proxyService, err := proxy.New(proxy.Config{
		Logger:              svc.Logger,
		MaxIdleConns:        transport.MaxIdleConns,
		IdleConnTimeout:     "90s",
		TLSHandshakeTimeout: "10s",
		Transport:           transport,
	})
	if err != nil {
		svc.Logger.Error("failed to create proxy service", "error", err.Error())
		return
	}

	authService, err := auth.New(auth.Config{
		Logger: svc.Logger,
		Keys:   svc.Keys,
	})
	if err != nil {
		svc.Logger.Error("failed to create auth service", "error", err.Error())
		return
	}

	// Create the main proxy handler that handles all gateway requests
	proxyHandler := &gateway_proxy.Handler{
		Logger:         svc.Logger,
		RoutingService: svc.RoutingService,
		Proxy:          proxyService,
		Auth:           authService,
		Validator:      svc.Validation,
	}

	// Create a mux for routing
	mux := http.NewServeMux()

	// Health check endpoint - only on main domain
	mux.HandleFunc("/unkey/_internal/liveness", func(w http.ResponseWriter, r *http.Request) {
		// Extract host without port
		host := r.Host
		if idx := strings.Index(host, ":"); idx != -1 {
			host = host[:idx]
		}

		if svc.MainDomain != "" && host != svc.MainDomain {
			http.NotFound(w, r)
			return
		}

		// Return health status
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"gateway"}`))
	})

	// HTTP server configuration for ACME challenges
	if serverType == HTTPServer {
		acmeHandler := &acme_challenge.Handler{
			Logger:         svc.Logger,
			RoutingService: svc.RoutingService,
			AcmeClient:     svc.AcmeClient,
		}

		// ACME challenge endpoint
		mux.Handle("/.well-known/acme-challenge/", srv.WrapHandler(acmeHandler.Handle, defaultMiddlewares))

		// Redirect all other HTTP traffic to HTTPS
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// Build HTTPS URL
			httpsURL := "https://" + r.Host + r.URL.RequestURI()
			// Use 307 to preserve method and body
			http.Redirect(w, r, httpsURL, http.StatusTemporaryRedirect)
		})
	}

	// HTTPS server configuration for main gateway
	if serverType == HTTPSServer {
		// All other routes go to the proxy handler
		mux.Handle("/", srv.WrapHandler(proxyHandler.Handle, defaultMiddlewares))
	}

	// Set the mux as the server handler
	srv.SetHandler(mux)
}
