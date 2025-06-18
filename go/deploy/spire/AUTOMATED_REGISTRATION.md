# SPIRE Automated Registration

This document describes the automated agent and service registration process for SPIRE.

## Quick Start

```bash
# Install and start SPIRE server
make install-server
make service-start-server

# Bootstrap and register agent (automated)
make bootstrap-agent
make register-agent

# Register all services
make register-services
```

## Automated Agent Registration

The `register-agent` command now provides a fully automated workflow:

1. **Automatic Join Token Generation**: Creates a join token with proper SPIFFE ID
2. **Systemd Integration**: Configures the agent with the join token via systemd drop-in
3. **Auto-Registration**: Agent automatically registers on first start
4. **Token Cleanup**: Removes join token after successful registration
5. **Re-registration Support**: Handles existing agents gracefully

### Usage

```bash
# First time registration
make register-agent

# Re-run safely (detects existing registration)
make register-agent
```

## Service Registration

The `register-services` command registers all Unkey services:

- metald
- billaged  
- builderd
- assetmanagerd

Each service gets:
- Unique SPIFFE ID: `spiffe://<trust-domain>/service/<name>`
- Unix path and user selectors for attestation
- 1-hour certificate lifetime

### Usage

```bash
# Register all services at once
make register-services

# Services automatically receive certificates when started
```

## Environment Support

Use `SPIRE_ENVIRONMENT` to deploy different configurations:

```bash
# Development (default)
SPIRE_ENVIRONMENT=dev make install-server

# Canary
SPIRE_ENVIRONMENT=canary make install-server

# Production
SPIRE_ENVIRONMENT=prod make install-server
```

## Trust Domains

- Development: `development.unkey.app`
- Canary: `canary.unkey.app`
- Production: `prod.unkey.app`

## Troubleshooting

```bash
# Check registration status
sudo /opt/spire/bin/spire-server entry show -socketPath /run/spire/server.sock

# View agent logs
sudo journalctl -u spire-agent -f

# Test agent SVID
sudo /opt/spire/bin/spire-agent api fetch x509 -socketPath /run/spire/sockets/agent.sock
```