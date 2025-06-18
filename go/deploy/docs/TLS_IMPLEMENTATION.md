# Opt-In TLS/SPIFFE Implementation

This PR introduces completely opt-in TLS support for all Unkey services, with zero disruption to existing deployments.

## What's Implemented

### 1. Unified TLS Provider (`pkg/tls`)
- Supports three modes: `disabled` (default), `file`, and `spiffe`
- Single interface for all TLS needs across services
- Graceful fallback if SPIFFE unavailable

### 2. SPIFFE Client (`pkg/spiffe`)
- Workload API integration for automatic mTLS
- No certificate files needed
- Automatic rotation every hour

### 3. Service Integration
- ✅ metald - Full TLS support added
- 🚧 billaged - Ready to implement
- 🚧 builderd - Ready to implement
- 🚧 assetmanagerd - Ready to implement

## How It Works

### Default Behavior (No Changes)
```bash
# Services run exactly as before - no TLS
./metald
```

### Opt-In to SPIFFE
```bash
# Enable with environment variable
UNKEY_METALD_TLS_MODE=spiffe ./metald

# Or use file-based TLS
UNKEY_METALD_TLS_MODE=file \
UNKEY_METALD_TLS_CERT_FILE=/path/to/cert.pem \
UNKEY_METALD_TLS_KEY_FILE=/path/to/key.pem \
./metald
```

## Configuration

Each service now supports optional TLS configuration:

```yaml
# Environment Variables
UNKEY_METALD_TLS_MODE=disabled|file|spiffe
UNKEY_METALD_TLS_CERT_FILE=/path/to/cert.pem
UNKEY_METALD_TLS_KEY_FILE=/path/to/key.pem
UNKEY_METALD_TLS_CA_FILE=/path/to/ca.pem
UNKEY_METALD_SPIFFE_SOCKET=/run/spire/sockets/agent.sock
```

## Key Features

1. **Zero Breaking Changes** - Services default to current behavior
2. **Gradual Rollout** - Enable per-service, per-environment
3. **Mixed Mode** - TLS and non-TLS services work together
4. **Easy Rollback** - Just set `TLS_MODE=disabled`

## Testing

```bash
# Test without TLS (default)
cd metald && go build -o build/metald ./cmd/api
./build/metald

# Test with mock SPIFFE
cd spire/quickstart && ./setup.sh
UNKEY_METALD_TLS_MODE=spiffe ./build/metald
```

## Next Steps

1. Apply same pattern to other services
2. Add integration tests
3. Deploy SPIRE infrastructure
4. Enable service-by-service in production

## Files Changed

### New Files
- `pkg/tls/provider.go` - Unified TLS provider
- `pkg/spiffe/client.go` - SPIFFE integration
- `spire/` - Complete SPIFFE/SPIRE documentation and configs

### Modified Files
- `metald/internal/config/config.go` - Added TLS configuration
- `metald/cmd/api/main.go` - Integrated TLS provider
- `metald/internal/billing/client.go` - Added TLS support
- `metald/internal/assetmanager/client.go` - Added TLS support

## Safety

- Services continue working without any configuration changes
- TLS failures log warnings but don't crash services
- Can be enabled/disabled with single env var
- No certificate files required with SPIFFE mode