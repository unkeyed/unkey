# SPIFFE/SPIRE Setup for Metald Example Client

This guide explains how to configure SPIRE to allow the example client to communicate with metald using mTLS.

## Prerequisites

1. SPIRE server and agent must be running
2. The SPIRE agent must be registered on the node
3. Metald service must be registered with SPIRE

## Quick Setup

```bash
# Register the client with SPIRE
./register-with-spire.sh

# Build and run the client
make build
./run-with-spiffe.sh
```

## Detailed Setup

### 1. Verify SPIRE Installation

```bash
# Check SPIRE server
systemctl status spire-server

# Check SPIRE agent
systemctl status spire-agent

# Verify agent socket exists
ls -la /run/spire/sockets/agent.sock
```

### 2. Register the Client

The example client needs a SPIFFE identity to communicate with metald. There are two registration modes:

#### Production Mode (Recommended)
Registers with strict selectors based on user and binary path:

```bash
# Register with default settings
./register-with-spire.sh

# Or explicitly specify production mode
./register-with-spire.sh prod
```

This creates an entry with:
- SPIFFE ID: `spiffe://development.unkey.app/client/metald-example-client`
- Selectors:
  - `unix:user:YOUR_USERNAME` - Must run as your user
  - `unix:path:/full/path/to/metald-client` - Must be the exact binary

#### Development Mode
More permissive registration for testing:

```bash
./register-with-spire.sh dev
```

This creates an entry with:
- SPIFFE ID: `spiffe://development.unkey.app/client/metald-example-client-dev`
- Selector: `unix:uid:YOUR_UID` - Any process with your UID

### 3. Verify Registration

```bash
# List all registered entries
sudo /opt/spire/bin/spire-server entry show -socketPath /run/spire/server.sock

# Check specific entry
sudo /opt/spire/bin/spire-server entry show \
    -socketPath /run/spire/server.sock \
    -spiffeID spiffe://development.unkey.app/client/metald-example-client
```

### 4. Test the Client

```bash
# Build the client
make build

# Run with SPIFFE
./metald-client \
    --addr https://localhost:8080 \
    --customer example-customer \
    --tls-mode spiffe \
    --action list
```

## How It Works

### SPIFFE Identity Flow

1. **Client Startup**: When the client starts with `--tls-mode spiffe`, it:
   - Connects to SPIRE agent via Unix socket
   - Requests its X.509 SVID (SPIFFE Verifiable Identity Document)
   - Agent verifies the client matches registered selectors

2. **Workload Attestation**: SPIRE agent checks:
   - Process user matches `unix:user` selector
   - Binary path matches `unix:path` selector
   - If both match, issues the SPIFFE identity

3. **mTLS Connection**: The client uses the SVID to:
   - Present its identity to metald
   - Verify metald's identity
   - Establish mutual TLS connection

### Trust Domain Verification

Both client and server verify they're in the same trust domain:
- Client: `spiffe://development.unkey.app/client/metald-example-client`
- Server: `spiffe://development.unkey.app/service/metald`

The shared trust domain `development.unkey.app` ensures they trust each other.

## Troubleshooting

### Client Gets "No Such Process" Error

```
Failed to create TLS provider: init SPIFFE: create X509 source: no such process
```

**Cause**: Client doesn't match registered selectors.

**Solution**:
1. Verify you're running as the correct user
2. Check binary path matches registration
3. Use `register-with-spire.sh dev` for testing

### Permission Denied on Socket

```
Failed to create TLS provider: init SPIFFE: create X509 source: permission denied
```

**Cause**: User doesn't have access to SPIRE agent socket.

**Solution**:
```bash
# Add user to spire group
sudo usermod -a -G spire $USER

# Re-login for group change to take effect
```

### No SVID Issued

```
Failed to create TLS provider: init SPIFFE: get SVID: no SVID issued
```

**Cause**: No matching registration entry.

**Solution**:
1. Run `./register-with-spire.sh`
2. Verify registration with entry show command
3. Check SPIRE agent logs: `journalctl -u spire-agent -f`

### Connection Refused

```
Failed to create VM: unavailable: connection refused
```

**Cause**: Metald not running or not listening on HTTPS.

**Solution**:
1. Check metald status: `systemctl status metald`
2. Verify metald TLS configuration
3. Check metald logs for TLS errors

## Custom Trust Domains

To use a different trust domain:

```bash
# Set trust domain before registration
export TRUST_DOMAIN=prod.unkey.app
./register-with-spire.sh

# Ensure metald uses same trust domain
UNKEY_METALD_TRUST_DOMAIN=prod.unkey.app metald
```

## Security Best Practices

1. **Use Production Mode**: Always use strict selectors in production
2. **Limit Permissions**: Run client with minimal privileges
3. **Rotate SVIDs**: Default TTL is 1 hour, adjust as needed
4. **Monitor Logs**: Check SPIRE logs for attestation failures
5. **Separate Environments**: Use different trust domains for dev/staging/prod

## Integration with CI/CD

For automated testing:

```bash
# Register CI runner identity
sudo /opt/spire/bin/spire-server entry create \
    -socketPath /run/spire/server.sock \
    -parentID spiffe://development.unkey.app/agent/node1 \
    -spiffeID spiffe://development.unkey.app/ci/runner \
    -selector k8s:pod-label:app:ci-runner \
    -x509SVIDTTL 300  # 5 minute TTL for CI jobs
```

## Advanced Configuration

### Custom Socket Path

```bash
# Use non-default socket
UNKEY_METALD_SPIFFE_SOCKET=/custom/spire/agent.sock \
./metald-client --tls-mode spiffe
```

### Multiple Clients

Register different clients with unique identities:

```bash
# Register admin client
sudo /opt/spire/bin/spire-server entry create \
    -spiffeID spiffe://development.unkey.app/client/metald-admin \
    -selector unix:user:admin \
    -admin

# Register monitoring client  
sudo /opt/spire/bin/spire-server entry create \
    -spiffeID spiffe://development.unkey.app/client/metald-monitor \
    -selector unix:user:monitor \
    -selector unix:group:monitoring
```

## Clean Up

To remove client registration:

```bash
# Find entry ID
ENTRY_ID=$(sudo /opt/spire/bin/spire-server entry show \
    -socketPath /run/spire/server.sock \
    -spiffeID spiffe://development.unkey.app/client/metald-example-client \
    | grep "Entry ID" | awk '{print $3}')

# Delete entry
sudo /opt/spire/bin/spire-server entry delete \
    -socketPath /run/spire/server.sock \
    -entryID "$ENTRY_ID"
```