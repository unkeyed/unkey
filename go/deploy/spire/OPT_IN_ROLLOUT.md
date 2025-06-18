# Opt-In SPIFFE/SPIRE Rollout Strategy

## Overview

This approach allows SPIFFE/SPIRE to be completely opt-in with zero disruption to existing services. Services continue working exactly as they do today until you explicitly enable SPIFFE.

## Configuration Examples

### 1. Current State (No Changes Needed)
```yaml
# metald/config.yaml - Works exactly as today
bind_address: ":8080"
# No TLS config = no TLS (current behavior)
```

### 2. Enable SPIFFE for One Service
```yaml
# metald/config.yaml
bind_address: ":8080"
tls:
  mode: "spiffe"  # Opt-in to SPIFFE
  spiffe_socket_path: "/run/spire/sockets/agent.sock"
```

### 3. Gradual Rollout with Feature Flags
```yaml
# metald/config.yaml
bind_address: ":8080"
tls:
  mode: "${TLS_MODE:-disabled}"  # Environment variable with default
```

Then control per deployment:
```bash
# Development (default)
systemctl start metald

# Test SPIFFE in staging  
TLS_MODE=spiffe systemctl start metald

# Production still uses default
systemctl start metald
```

## Rollout Phases

### Phase 0: Add TLS Provider (No Impact)
- Merge the TLS provider code
- Update services to use provider with `disabled` default
- **Result**: No behavior change, services work exactly as before

### Phase 1: Dev Environment Testing
```bash
# Deploy SPIRE only to dev environment
docker-compose -f spire/docker-compose.yml up -d

# Enable for one service via env var
TLS_MODE=spiffe ./metald

# Other services unchanged
./billaged  # Still no TLS
```

### Phase 2: Staging Validation
```yaml
# staging/metald.env
TLS_MODE=spiffe

# staging/billaged.env  
TLS_MODE=disabled  # Explicit, but same as default
```

### Phase 3: Production Service-by-Service
```bash
# Week 1: Enable assetmanagerd (least critical)
kubectl set env deployment/assetmanagerd TLS_MODE=spiffe

# Week 2: Enable billaged
kubectl set env deployment/billaged TLS_MODE=spiffe

# Week 3: Enable builderd
kubectl set env deployment/builderd TLS_MODE=spiffe

# Week 4: Enable metald (most critical)
kubectl set env deployment/metald TLS_MODE=spiffe
```

## Rollback is One Command

If anything goes wrong:
```bash
# Instant rollback for any service
kubectl set env deployment/metald TLS_MODE=disabled

# Or just remove the env var to use default
kubectl set env deployment/metald TLS_MODE-
```

## Mixed Mode Operation

During rollout, services can communicate regardless of TLS mode:

```
metald (SPIFFE) → billaged (No TLS) ✓
builderd (No TLS) → assetmanagerd (SPIFFE) ✓  
```

The provider handles this automatically:
- SPIFFE services accept both SPIFFE and non-TLS connections
- Non-TLS services continue working as normal
- Client connections adapt to the target service

## Code Changes Required

### Minimal Service Update
```diff
// main.go
+ import "github.com/unkeyed/unkey/go/deploy/pkg/tls"

func main() {
    cfg := loadConfig()
    
+   // Create TLS provider (defaults to disabled)
+   tlsProvider, err := tls.NewProvider(ctx, cfg.TLS)
+   if err != nil {
+       log.Printf("TLS init failed, continuing without: %v", err)
+       tlsProvider, _ = tls.NewProvider(ctx, tls.Config{Mode: tls.ModeDisabled})
+   }
+   defer tlsProvider.Close()
    
    // Rest of your code unchanged
    server := &http.Server{
        Addr: cfg.BindAddress,
        Handler: mux,
    }
    
-   log.Fatal(server.ListenAndServe())
+   if tlsConfig, _ := tlsProvider.ServerTLSConfig(); tlsConfig != nil {
+       server.TLSConfig = tlsConfig
+       log.Fatal(server.ListenAndServeTLS("", ""))
+   } else {
+       log.Fatal(server.ListenAndServe())
+   }
}
```

### Config Structure
```diff
type Config struct {
    BindAddress string
    Database    DatabaseConfig
+   TLS         *tls.Config `json:"tls,omitempty"`
}
```

## Monitoring the Rollout

### 1. Service Start Logs
```
INFO: Starting with TLS mode: disabled (default)
INFO: Starting with TLS mode: spiffe (opted-in)
```

### 2. SPIFFE Metrics (only for opted-in services)
```
spiffe_svid_renewed_total{service="metald"} 24
spiffe_connection_authenticated{from="metald",to="billaged"} 1
```

### 3. Health Checks Work Regardless
```bash
# These work the same with or without TLS
curl http://metald:8080/health
curl https://metald:8080/health  # If SPIFFE enabled
```

## Developer Experience

### Local Development (No Changes)
```bash
# Just run services as normal
go run ./metald
go run ./billaged
```

### Testing with SPIFFE
```bash
# Start local SPIRE (optional)
cd spire/quickstart && ./setup.sh

# Run with SPIFFE
TLS_MODE=spiffe go run ./metald
```

### CI/CD (No Changes)
- Existing tests continue working
- Add optional SPIFFE integration tests
- No changes to build or deployment process

## Why This Approach Works

1. **Zero Disruption**: Default behavior unchanged
2. **Gradual Adoption**: Enable per-service, per-environment  
3. **Easy Rollback**: One config change or env var
4. **Mixed Mode**: Services work together regardless of TLS mode
5. **Developer Friendly**: No changes to dev workflow unless wanted

## Success Metrics

- Week 1: TLS provider merged, no services using it
- Week 2: One service in dev using SPIFFE
- Week 3: All services in staging capable of SPIFFE
- Week 4: First production service on SPIFFE
- Week 8: All production services on SPIFFE
- Week 12: Remove support for `disabled` mode

This gives you 3 months to gradually adopt SPIFFE with multiple rollback points and zero required disruption.