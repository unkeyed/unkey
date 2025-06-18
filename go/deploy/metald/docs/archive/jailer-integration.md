# Jailer Integration

> **Note**: This document describes the legacy external jailer approach. Metald now uses an **integrated jailer implementation** that provides better control and reliability.

## Current Implementation

Metald includes an integrated jailer implementation that replaces the external Firecracker jailer binary. This approach provides:

- Better network namespace integration
- Improved TAP device handling
- Direct control over privilege dropping
- Enhanced error reporting and observability

**See [Integrated Jailer Documentation](integrated-jailer.md) for the current implementation details.**

## Legacy External Jailer (Deprecated)

The information below describes the previous external jailer approach, which has been replaced by the integrated implementation.

### Why We Moved Away from External Jailer

1. **Network Namespace Issues**: The external jailer created TAP devices outside the network namespace, causing "device not found" errors
2. **Process Control**: Limited control over the forking and execution process
3. **Error Handling**: Poor error reporting from the external binary
4. **Integration**: Difficult to integrate with metald's networking and observability

### Migration

No action is required to migrate from the external jailer to the integrated implementation. The integrated jailer:

- Uses the same configuration parameters (UID, GID, chroot directory)
- Maintains the same security isolation guarantees
- Is fully backward compatible with existing deployments

### Removed Configuration

The following environment variables are no longer used:

- `UNKEY_METALD_JAILER_BINARY` - No longer needed
- `UNKEY_METALD_FIRECRACKER_BINARY` - Path is now hardcoded to `/usr/local/bin/firecracker`

All other jailer configuration remains the same.