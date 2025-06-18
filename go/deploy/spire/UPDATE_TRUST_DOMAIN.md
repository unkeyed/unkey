# Trust Domain Update: unkey.io → unkey.app

All SPIFFE trust domain references have been updated from `unkey.io` to `unkey.app`.

## Updated Files:

1. **Documentation**:
   - `spire/TRUST_DOMAIN_STRATEGY.md`
   - `spire/environments/README.md`

2. **SPIRE Configurations**:
   - `spire/quickstart/server/server.conf`
   - `spire/environments/dev/server.conf`
   - `spire/environments/canary/server.conf`
   - `spire/environments/prod/server.conf`

3. **Example Code**:
   - `spire/examples/trust-domain-demo.go`

## New Trust Domain Structure:

```
Development:  spiffe://dev.unkey.app
Canary:       spiffe://canary.unkey.app
Production:   spiffe://prod.unkey.app
```

## Important Notes:

- Trust domains do NOT need to be real DNS names
- They are just logical identifiers within SPIFFE/SPIRE
- Each environment maintains complete isolation
- Services can only communicate within their trust domain

## Next Steps:

When deploying SPIRE:
1. Use the appropriate environment configuration from `spire/environments/`
2. Register workloads with the matching trust domain
3. Services will automatically enforce trust domain boundaries