# SPIFFE Trust Domain Strategy for Unkey

## Understanding Trust Domains

SPIFFE trust domains are **NOT** DNS names - they're logical identifiers that:
- Group related workloads under a common trust boundary
- Enable authorization policies between services
- Isolate different environments cryptographically
- Follow reverse-DNS format by convention (but don't require DNS resolution)

## Recommended Trust Domain Structure

### Option 1: Environment-Based Separation (Recommended)
```
Development:  spiffe://dev.unkey.app
Canary:       spiffe://canary.unkey.app
Production:   spiffe://prod.unkey.app
```

**Pros:**
- Complete cryptographic isolation between environments
- No accidental cross-environment communication
- Clear security boundaries
- Simple mental model

**Cons:**
- Requires separate SPIRE deployments per environment
- No certificate sharing between environments

### Option 2: Single Trust Domain with Path-Based Environments
```
All environments: spiffe://unkey.app

With workload IDs like:
- spiffe://unkey.app/dev/service/metald
- spiffe://unkey.app/canary/service/metald  
- spiffe://unkey.app/prod/service/metald
```

**Pros:**
- Single SPIRE deployment can serve all environments
- Easier certificate management
- Can implement cross-environment communication if needed

**Cons:**
- Requires careful authorization policies
- Risk of misconfiguration allowing cross-environment access
- More complex ACLs

## Implementation Example

### 1. Environment-Based Configuration

```bash
# Development
export SPIFFE_TRUST_DOMAIN="dev.unkey.app"
export SPIFFE_WORKLOAD_ID="spiffe://dev.unkey.app/service/metald"

# Canary
export SPIFFE_TRUST_DOMAIN="canary.unkey.app"
export SPIFFE_WORKLOAD_ID="spiffe://canary.unkey.app/service/metald"

# Production
export SPIFFE_TRUST_DOMAIN="prod.unkey.app"
export SPIFFE_WORKLOAD_ID="spiffe://prod.unkey.app/service/metald"
```

### 2. Service Authorization

In `pkg/spiffe/client.go`, the authorization is already configured correctly:
```go
// Only accepts SVIDs from the same trust domain
tlsconfig.AuthorizeMemberOf(c.id.TrustDomain())
```

This means:
- Dev services only accept connections from `dev.unkey.app`
- Prod services only accept connections from `prod.unkey.app`
- Complete environment isolation by default

### 3. SPIRE Server Configuration

For each environment:

```hcl
# spire-server.conf
server {
    bind_address = "0.0.0.0"
    bind_port = "8081"
    trust_domain = "dev.unkey.app"  # Change per environment
    data_dir = "/opt/spire/data/server"
    log_level = "INFO"
}
```

### 4. Workload Registration

```bash
# Register metald in development
spire-server entry create \
    -spiffeID spiffe://dev.unkey.app/service/metald \
    -parentID spiffe://dev.unkey.app/agent/node1 \
    -selector k8s:ns:unkey \
    -selector k8s:pod-label:app:metald

# Register billaged in development  
spire-server entry create \
    -spiffeID spiffe://dev.unkey.app/service/billaged \
    -parentID spiffe://dev.unkey.app/agent/node1 \
    -selector k8s:ns:unkey \
    -selector k8s:pod-label:app:billaged
```

## Migration Path

### Phase 1: Development Environment
1. Deploy SPIRE with `dev.unkey.app` trust domain
2. Enable SPIFFE mode for non-critical services
3. Monitor and validate mTLS connectivity

### Phase 2: Canary Environment
1. Deploy separate SPIRE with `canary.unkey.app`
2. Roll out to canary services
3. Validate isolation from dev environment

### Phase 3: Production Environment
1. Deploy production SPIRE with `prod.unkey.app`
2. Gradual rollout with feature flags
3. Monitor performance and reliability

## DNS Clarification

The trust domains (`dev.unkey.app`, `prod.unkey.app`) do **NOT** need to:
- Resolve in DNS
- Have A/AAAA records
- Be registered domains
- Be publicly accessible

They are purely logical identifiers used within the SPIFFE/SPIRE system.

## Security Considerations

1. **Environment Isolation**: Different trust domains ensure complete cryptographic isolation
2. **Rotation**: SPIFFE handles automatic hourly certificate rotation
3. **Revocation**: Immediate revocation by removing workload registrations
4. **Auditability**: All certificate issuance logged by SPIRE server

## Monitoring

Track these metrics per environment:
- Certificate issuance rate
- Failed authentication attempts
- Certificate expiration warnings
- Cross-environment connection attempts (should be zero)

## Recommendations

1. **Use environment-based trust domains** for maximum security
2. **Start with development** to gain operational experience
3. **Automate workload registration** as part of deployment pipeline
4. **Monitor trust domain boundaries** to detect misconfigurations
5. **Plan for disaster recovery** with SPIRE server backup procedures