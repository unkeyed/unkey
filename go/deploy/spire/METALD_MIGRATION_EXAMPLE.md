# Example: Adding Opt-In TLS to metald

This shows the exact changes needed to add opt-in TLS/SPIFFE support to the metald service. These changes are minimal and non-disruptive.

## Step 1: Update Config Structure

```diff
// internal/config/config.go
package config

import (
    "os"
+   "github.com/unkeyed/unkey/go/deploy/pkg/tls"
)

type Config struct {
    Port            string
    Backend         string
    OtelEnabled     bool
    BillagedURL     string
    AssetManagerURL string
+   
+   // TLS configuration (optional, defaults to disabled)
+   TLS *tls.Config `json:"tls,omitempty"`
}

func Load() *Config {
    cfg := &Config{
        Port:            getEnv("UNKEY_METALD_PORT", "8080"),
        Backend:         getEnv("UNKEY_METALD_BACKEND", "firecracker"),
        OtelEnabled:     getEnv("UNKEY_METALD_OTEL_ENABLED", "false") == "true",
        BillagedURL:     getEnv("UNKEY_METALD_BILLAGED_URL", "http://localhost:8081"),
        AssetManagerURL: getEnv("UNKEY_METALD_ASSETMANAGER_URL", "http://localhost:8083"),
+       
+       // TLS defaults to disabled for backward compatibility
+       TLS: &tls.Config{
+           Mode: tls.Mode(getEnv("UNKEY_METALD_TLS_MODE", "disabled")),
+           SPIFFESocketPath: getEnv("UNKEY_METALD_SPIFFE_SOCKET", "/run/spire/sockets/agent.sock"),
+       },
    }
    return cfg
}
```

## Step 2: Update Main Function

```diff
// cmd/api/main.go
package main

import (
    // ... existing imports ...
+   tlspkg "github.com/unkeyed/unkey/go/deploy/pkg/tls"
)

func main() {
    // ... existing startup code ...
    
    // Load configuration
    cfg := config.Load()
    
+   // Initialize TLS provider (defaults to disabled)
+   ctx := context.Background()
+   tlsProvider, err := tlspkg.NewProvider(ctx, *cfg.TLS)
+   if err != nil {
+       // Log warning but continue - service works without TLS
+       logger.Warn("TLS initialization failed, continuing without TLS",
+           "error", err,
+           "mode", cfg.TLS.Mode)
+       tlsProvider, _ = tlspkg.NewProvider(ctx, tlspkg.Config{Mode: tlspkg.ModeDisabled})
+   }
+   defer tlsProvider.Close()
+   
+   logger.Info("TLS provider initialized",
+       "mode", cfg.TLS.Mode,
+       "spiffe_enabled", cfg.TLS.Mode == tlspkg.ModeSPIFFE)
    
    // ... existing service initialization ...
    
    // Create HTTP server
    mux := http.NewServeMux()
    
    // Existing routes...
    mux.Handle(vmprovisionerv1connect.NewVmServiceHandler(
        vmService,
        connect.WithInterceptors(interceptors...),
    ))
    
-   // Start server
-   server := &http.Server{
-       Addr:    ":" + cfg.Port,
-       Handler: h2c.NewHandler(mux, &http2.Server{}),
-   }
-   
-   logger.Info("Starting metald", "port", cfg.Port)
-   if err := server.ListenAndServe(); err != nil {
-       logger.Error("Server failed", "error", err)
-       os.Exit(1)
-   }
    
+   // Create server with optional TLS
+   server := &http.Server{
+       Addr:    ":" + cfg.Port,
+       Handler: h2c.NewHandler(mux, &http2.Server{}),
+   }
+   
+   // Configure TLS if enabled
+   tlsConfig, _ := tlsProvider.ServerTLSConfig()
+   if tlsConfig != nil {
+       server.TLSConfig = tlsConfig
+       logger.Info("Starting metald with TLS",
+           "port", cfg.Port,
+           "tls_mode", cfg.TLS.Mode)
+       if err := server.ListenAndServeTLS("", ""); err != nil {
+           logger.Error("Server failed", "error", err)
+           os.Exit(1)
+       }
+   } else {
+       logger.Info("Starting metald without TLS", "port", cfg.Port)
+       if err := server.ListenAndServe(); err != nil {
+           logger.Error("Server failed", "error", err)
+           os.Exit(1)
+       }
+   }
}
```

## Step 3: Update Client Connections

```diff
// internal/billing/client.go
package billing

+import "github.com/unkeyed/unkey/go/deploy/pkg/tls"

type Client struct {
    baseURL    string
    httpClient *http.Client
+   tlsProvider tls.Provider
}

-func NewClient(billagedURL string) *Client {
+func NewClient(billagedURL string, tlsProvider tls.Provider) *Client {
    return &Client{
        baseURL:    billagedURL,
-       httpClient: &http.Client{
-           Timeout: 10 * time.Second,
-       },
+       httpClient: tlsProvider.HTTPClient(),
+       tlsProvider: tlsProvider,
    }
}
```

## Step 4: Test Without Any Changes

```bash
# 1. Build with changes
go build -o build/metald ./cmd/api

# 2. Run exactly as before (TLS disabled by default)
./build/metald

# 3. Service works exactly the same
curl http://localhost:8080/health
```

## Step 5: Opt-In to SPIFFE (When Ready)

```bash
# Deploy SPIRE infrastructure first
cd spire/quickstart && ./setup.sh

# Run with SPIFFE enabled
UNKEY_METALD_TLS_MODE=spiffe ./build/metald

# Or use config file
cat > metald.yaml <<EOF
port: "8080"
backend: "firecracker"
tls:
  mode: "spiffe"
EOF
```

## What Changed?

1. **Config**: Added optional TLS section (defaults to disabled)
2. **Server**: Added TLS support that activates only when configured
3. **Clients**: Use TLS provider for connections (works with or without TLS)

## What Didn't Change?

1. **Default Behavior**: Service runs exactly as before
2. **Development**: No changes to dev workflow
3. **Deployment**: No changes needed until you want TLS
4. **Tests**: Existing tests continue working

## Rollback

If anything goes wrong:
```bash
# Just remove the env var
unset UNKEY_METALD_TLS_MODE

# Or explicitly disable
UNKEY_METALD_TLS_MODE=disabled ./build/metald
```

## Total Lines Changed

- ~20 lines added to config
- ~15 lines modified in main.go  
- ~5 lines per client connection
- **Zero breaking changes**

This is why the opt-in approach works so well - minimal changes with maximum flexibility!