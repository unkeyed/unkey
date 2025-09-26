# SPIRE: Secure Service Identity for Unkey Deploy

## What is SPIRE?

SPIRE (SPIFFE Runtime Environment) is the production-ready implementation of SPIFFE that provides automatic, cryptographically-verifiable service identities. It eliminates the need for manual certificate management in our microservices architecture.

## The Problem SPIRE Solves

### Traditional Certificate Management Pain Points
- **Manual certificate distribution**: Copying cert files to each service
- **Certificate rotation nightmares**: Expired certs breaking production at 3am
- **Security risks**: Long-lived certificates sitting in files
- **Operational overhead**: Complex automation scripts for cert lifecycle
- **Trust boundaries**: Hard to verify which service is really calling you

### How SPIRE Eliminates These Problems
- **Zero certificate files**: Everything happens in memory via APIs
- **Automatic rotation**: Certificates refresh every hour automatically
- **Strong workload identity**: Services proven by process attestation, not just file possession
- **Dynamic trust**: Policy-based access control with runtime verification
- **Simplified operations**: Deploy once, identities work forever

## Architecture Overview

```
┌─────────────────────┐
│   SPIRE Server      │  ← Central trust authority
│  (Trust Root CA)    │  ← Issues short-lived certificates
└──────────┬──────────┘  ← Manages service registrations
           │
    ┌──────┴──────┐
    │             │
┌───▼───┐    ┌───▼───┐
│ Host 1 │    │ Host 2 │
├────────┤    ├────────┤
│ Agent  │    │ Agent  │  ← Verifies workload identity
├────────┤    ├────────┤  ← Delivers certificates to services
│metald  │    │builderd│  ← Services get automatic mTLS
│billaged│    │assetmgr│  ← No certificate files needed
└────────┘    └────────┘
```

## How It Fits Into Our Architecture

### Service Communication Flow
1. **Service Startup**: Each service (metald, billaged, etc.) contacts local SPIRE agent
2. **Identity Verification**: Agent verifies service identity using process attestation
3. **Certificate Delivery**: Agent provides short-lived X.509 certificate with SPIFFE ID
4. **Secure Communication**: Services use certificates for automatic mTLS
5. **Automatic Renewal**: Certificates refresh every hour without service restart

### Integration with Unkey Services

#### Core Services Using SPIRE
- **metald**: VM management service with identity `spiffe://production.unkey.cloud/service/metald`
- **billaged**: Billing aggregation service with identity `spiffe://production.unkey.cloud/service/billaged`
- **builderd**: Container build service with identity `spiffe://production.unkey.cloud/service/builderd`
- **assetmanagerd**: Asset management service with identity `spiffe://production.unkey.cloud/service/assetmanagerd`

#### Multi-Tenant Identity Patterns
```
# Service-level identity (most common)
spiffe://production.unkey.cloud/service/metald

# Customer-scoped identity (for VM processes)
spiffe://production.unkey.cloud/service/metald/customer/cust-123

# Tenant-scoped identity (for build isolation)
spiffe://production.unkey.cloud/service/builderd/tenant/acme-corp
```

## Environment Isolation Strategy

### Separate Trust Domains
Each environment has its own trust domain for complete cryptographic isolation:

- **Development**: `spiffe://development.unkey.cloud` - Fast iteration, verbose logging
- **Canary**: `spiffe://canary.unkey.cloud` - Production-like testing environment
- **Production**: `spiffe://production.unkey.cloud` - Hardened configuration, HA deployment

### Why Separate Trust Domains?
1. **Security**: Services in different environments cannot communicate even if misconfigured
2. **Clarity**: Easy to identify which environment a certificate belongs to
3. **Compliance**: Clear security boundaries for audit purposes
4. **Simplicity**: No complex ACL rules - trust domain provides natural isolation

## Key Benefits for Developers

### Before SPIRE: Manual Certificate Hell
```go
// Old way - manual certificate management
cert, err := tls.LoadX509KeyPair("/etc/service/cert.pem", "/etc/service/key.pem")
if err != nil {
    log.Fatal("Certificate file missing or expired!")
}

client := &http.Client{
    Transport: &http.Transport{
        TLSClientConfig: &tls.Config{
            Certificates: []tls.Certificate{cert},
            RootCAs:      loadCA("/etc/service/ca.pem"),
        },
    },
}
```

### After SPIRE: Automatic Everything
```go
// New way - automatic identity and mTLS
spiffeClient, err := spiffe.New(ctx)
if err != nil {
    log.Fatal("SPIRE agent not available")
}

// HTTP client with automatic mTLS - no certificates!
client := spiffeClient.HTTPClient()
resp, err := client.Get("https://billaged:8081/api/usage")
```

### Operational Benefits
- **No cert files to manage**: Everything handled in memory
- **No rotation scripts**: Certificates refresh automatically every hour
- **No midnight pages**: Expired certificates can't break production
- **Strong security**: Process-based attestation ensures service authenticity
- **Better debugging**: Full audit trail of all service communications

## Security Model

### Workload Attestation
SPIRE verifies service identity using multiple factors:
- **Process path**: `/usr/bin/metald`
- **User/group**: `unkey-metald:unkey-metald`
- **Systemd unit**: `metald.service`
- **Cgroup hierarchy**: Systemd-managed processes

### Certificate Lifecycle
- **TTL**: 1 hour (production), 5 minutes (development)
- **Rotation**: Automatic renewal at 50% of TTL
- **Revocation**: Immediate when workload stops or registration changes
- **Validation**: Continuous verification of workload identity

### Trust Bundle Management
- **Root CA**: Managed by SPIRE server with 1-year TTL
- **Intermediate CAs**: Automatic rotation for zero-downtime updates
- **Cross-environment isolation**: Separate CAs per trust domain

## Production Deployment Considerations

### High Availability
- **SPIRE Server**: Deploy in HA mode with shared database
- **Database**: PostgreSQL with replication for registration data
- **Key Management**: AWS KMS for hardware-backed key storage
- **Monitoring**: Prometheus metrics for certificate issuance and rotation

### Scaling
- **SPIRE Agents**: One per host/node, lightweight resource usage
- **Certificate caching**: Agent caches certificates locally
- **Registration entries**: Centrally managed via SPIRE server API
- **Backup/Recovery**: Database backups include all registration state

## Quick Reference

### Common Commands
```bash
# Check service SVID
sudo -u unkey-metald spire-agent api fetch x509 -socketPath /run/spire/sockets/agent.sock

# Register new service
spire-server entry create \
  -spiffeID spiffe://production.unkey.cloud/service/newservice \
  -parentID spiffe://production.unkey.cloud/agent/server \
  -selector unix:path:/usr/bin/newservice \
  -selector unix:user:unkey-newservice

# View all registrations
spire-server entry show
```

### Directory Structure
```
spire/
├── environments/           # Per-environment configurations
│   ├── dev/               # Development settings
│   ├── canary/            # Canary environment
│   └── prod/              # Production configuration
├── agent/                 # Agent configuration templates
├── contrib/               # Systemd units and helpers
├── scripts/               # Automation and setup scripts
└── docs/                  # This documentation
```

## Related Documentation

- [Understanding SPIFFE](./UNDERSTANDING_SPIFFE.md) - Developer guide to SPIFFE concepts
- [Architecture Details](./architecture.md) - Technical implementation details
- [Environment Configurations](../environments/README.md) - Per-environment setup guide

## Further Reading

- **SPIFFE Specification**: https://spiffe.io/docs/latest/spiffe/
- **SPIRE Documentation**: https://spiffe.io/docs/latest/spire/
- **Production Deployment Guide**: https://spiffe.io/docs/latest/planning/production/
- **Community Support**: https://spiffe.slack.com
