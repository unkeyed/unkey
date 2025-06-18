# SPIFFE/SPIRE for Unkey Services

This directory contains everything needed to implement opt-in SPIFFE/SPIRE for zero-trust service authentication.

## 🎯 Key Design Principle: Fully Opt-In

**No disruption to existing services**. Everything continues working exactly as it does today until you explicitly enable SPIFFE.

## 📁 What's Here

### Core Implementation
- `pkg/tls/provider.go` - Unified TLS provider supporting disabled/file/SPIFFE modes
- `pkg/spiffe/client.go` - SPIFFE workload API integration

### Documentation
- `UNDERSTANDING_SPIFFE.md` - Developer-friendly explanation of SPIFFE concepts
- `DEPLOYMENT_PLAN.md` - Complete 5-week rollout strategy
- `OPT_IN_ROLLOUT.md` - How to enable SPIFFE without disruption
- `METALD_MIGRATION_EXAMPLE.md` - Concrete example for metald service

### Configuration
- `server/` - SPIRE server configuration
- `agent/` - SPIRE agent configuration  
- `registration/` - Service registration scripts
- `architecture.md` - System design and identity scheme

### Quick Start
- `quickstart/` - Docker Compose environment for local testing

## 🚀 Getting Started

### 1. Test Locally (No Code Changes)
```bash
cd quickstart
./setup.sh
docker-compose logs -f
```

### 2. Add to a Service (Minimal Changes)
See `METALD_MIGRATION_EXAMPLE.md` for exact diffs needed (~35 lines).

### 3. Enable in Development
```bash
# Default (no TLS)
./metald

# With SPIFFE
UNKEY_METALD_TLS_MODE=spiffe ./metald
```

### 4. Production Rollout
Follow `OPT_IN_ROLLOUT.md` for safe, gradual deployment.

## 🔄 Migration Path

```
Current State → Add Provider → Test in Dev → Enable in Staging → Production Service by Service
     (0)           (1)            (2)             (3)                    (4)
```

Each step is reversible with one environment variable change.

## 🎭 How It Works

1. **No SPIFFE**: Services use HTTP (current state)
2. **SPIFFE Available**: Services get automatic mTLS
3. **Mixed Mode**: SPIFFE and non-SPIFFE services work together

## 📊 Benefits When Enabled

- **Zero certificate management** - No files, no rotation
- **1-hour certificate lifetime** - Limited blast radius
- **Service identity** - Not just "has a certificate"
- **Automatic rotation** - No 3am cert expiry alerts

## ⚡ Quick Commands

```bash
# Check if service can get SPIFFE identity
spire-agent api fetch x509 -socketPath /run/spire/sockets/agent.sock

# View registered services
spire-server entry list

# Monitor certificate rotation
watch -n 10 'spire-agent api fetch x509 | grep "SPIFFE ID"'
```

## 🔗 Resources

- [SPIFFE Concepts](https://spiffe.io/docs/latest/spiffe-about/spiffe-concepts/)
- [SPIRE Quickstart](https://spiffe.io/docs/latest/try/getting-started-linux-macos-x86/)
- [Production Guide](https://spiffe.io/docs/latest/planning/production/)

---

**Remember**: This is completely opt-in. Nothing changes until you're ready.