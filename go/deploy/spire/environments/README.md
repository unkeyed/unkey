# SPIRE Environment Configurations

This directory contains SPIRE configurations for each environment, implementing our trust domain isolation strategy.

## Trust Domain Strategy

Each environment has its own trust domain to ensure complete cryptographic isolation:

- **Development**: `spiffe://development.unkey.cloud`
- **Canary**: `spiffe://canary.unkey.cloud`
- **Production**: `spiffe://production.unkey.cloud`

## Why Separate Trust Domains?

1. **Security**: Services in different environments cannot communicate, even if misconfigured
2. **Clarity**: Easy to identify which environment a certificate belongs to
3. **Compliance**: Clear security boundaries for audit purposes
4. **Simplicity**: No complex ACL rules needed - trust domain provides isolation

## Directory Structure

```
environments/
├── dev/
│   ├── server.conf      # SPIRE server config for dev
│   ├── agent.conf       # SPIRE agent config for dev
│   └── registrations/   # Workload registrations for dev
├── canary/
│   ├── server.conf      # SPIRE server config for canary
│   ├── agent.conf       # SPIRE agent config for canary
│   └── registrations/   # Workload registrations for canary
└── prod/
    ├── server.conf      # SPIRE server config for production
    ├── agent.conf       # SPIRE agent config for production
    └── registrations/   # Workload registrations for production
```

## Deployment

Each environment should have its own SPIRE deployment:

```bash
# Development
kubectl apply -f environments/dev/

# Canary
kubectl apply -f environments/canary/

# Production
kubectl apply -f environments/prod/
```

## Service Names

Services keep the same logical names across environments:
- `metald`
- `billaged`
- `builderd`
- `assetmanagerd`

The full SPIFFE ID includes the environment via trust domain:
- Dev: `spiffe://development.unkey.cloud/service/metald`
- Prod: `spiffe://production.unkey.cloud/service/metald`

## Note on DNS

The trust domains (dev.unkey.cloud, production.unkey.cloud) do **NOT** need to be real DNS names. They are just identifiers used within SPIFFE/SPIRE.
