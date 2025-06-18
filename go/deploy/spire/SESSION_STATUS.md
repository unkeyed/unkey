# SPIRE Configuration Session Status - 2025-06-16

## What We Fixed Today

### Major Issues Resolved
1. **Removed quickstart directory** - Was causing trust domain conflicts
2. **Fixed port inconsistencies** - Standardized on:
   - Server API: 8081
   - Health checks: 8080 
   - Agent health: 8082
   - Prometheus metrics: 9988 (server), 9989 (agent)
3. **Fixed trust domain mismatches** - Now using environment variables consistently
4. **Secured all bind addresses** - Changed from 0.0.0.0 to 127.0.0.1
5. **Fixed systemd paths** - Now using /usr/local/bin/spire-{server,agent}
6. **Updated to SPIRE 1.12.2** - Latest stable version

### Configuration Structure
```
spire/
├── server/spire-server.conf      # Base template with env vars
├── agent/spire-agent.conf        # Agent config with proper paths
├── environments/
│   ├── dev/server.conf          # SQLite, debug logging, 5m TTLs
│   ├── canary/server.conf       # PostgreSQL, isolated trust domain
│   └── prod/server.conf         # PostgreSQL, AWS KMS, IAM roles
├── systemd/
│   ├── spire-server.service     # Fixed paths, validation, security
│   └── spire-agent.service      # Proper permissions, groups
└── scripts/
    └── setup-spire.sh           # Full installation script
```

### Key Environment Variables
- `UNKEY_SPIRE_TRUST_DOMAIN` - dev.unkey.app / canary.unkey.app / prod.unkey.app
- `UNKEY_SPIRE_LOG_LEVEL` - DEBUG / INFO
- `UNKEY_SPIRE_DB_TYPE` - sqlite3 / postgres
- `UNKEY_SPIRE_DB_CONNECTION` - Database connection string
- `UNKEY_SPIRE_SERVER_ADDRESS` - Server address for agents

## Next Steps for Tomorrow

### 1. Bootstrap SPIRE Infrastructure
```bash
# Install SPIRE on a test machine
sudo ./spire/scripts/setup-spire.sh dev

# Start the server
sudo systemctl start spire-server
sudo systemctl status spire-server

# Check server health
curl http://localhost:8080/live
```

### 2. Generate Trust Bundle for Agents
```bash
# Get the server's CA certificate
sudo spire-server bundle show -socketPath /run/spire/server.sock > /etc/spire/agent/bundle.crt

# This bundle needs to be distributed to all agents
```

### 3. Create Initial Join Token
```bash
# Generate a join token for the first agent
sudo spire-server token generate -spiffeID spiffe://dev.unkey.app/agent/node1 -ttl 600
```

### 4. Register Workloads
Look at `spire/registration/services.sh` for examples. Each service needs:
- SPIFFE ID: `spiffe://dev.unkey.app/service/metald`
- Selectors: path, user, systemd unit
- Parent ID: The agent's SPIFFE ID

### 5. Test Service Integration
The services (metald, assetmanagerd, etc.) already have SPIFFE support via:
- `pkg/tls/provider.go` - Unified TLS provider
- `pkg/spiffe/client.go` - SPIFFE client implementation
- Environment variable: `UNKEY_*_TLS_MODE=spiffe`

### 6. Production Considerations
- Set up PostgreSQL database for SPIRE server
- Create AWS KMS keys for key management
- Configure IAM roles for EC2 attestation
- Set up monitoring/alerting on metrics
- Plan gradual rollout using enable/disable scripts

### Quick Test Commands
```bash
# Check SPIRE server logs
sudo journalctl -u spire-server -f

# List all registered entries
sudo spire-server entry list -socketPath /run/spire/server.sock

# Check agent status
sudo spire-agent healthcheck -socketPath /run/spire/agent.sock
```

## Important Notes
- All configs now follow UNKEY_* environment variable naming
- Security hardening is enabled in systemd units
- Audit logging is enabled even in dev for debugging
- Trust domains are environment-isolated (dev/canary/prod)
- No more hardcoded secrets - everything uses env vars

The SPIRE setup is now clean, secure, and ready for testing!