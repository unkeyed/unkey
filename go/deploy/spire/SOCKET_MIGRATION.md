# SPIRE Socket Migration Documentation

## Problem
- Systemd's security features (PrivateTmp, ProtectSystem) create namespace isolation
- Sockets in `/run/spire` are not visible outside the service's namespace
- This breaks inter-service communication

## Solutions Evaluated

### 1. Abstract Unix Sockets (@spire-server)
- **Pros**: No filesystem presence, no namespace issues
- **Cons**: SPIRE CLI tools don't handle them properly
- **Status**: Server supports it, but client tools fail

### 2. /var/lib/spire Sockets
- **Pros**: Persistent location, works with all tools
- **Cons**: Less secure than /run, needs cleanup on start
- **Status**: Currently implemented

### 3. Bind Mounts with Security
- **Pros**: Keeps /run location, maintains security
- **Cons**: Complex systemd configuration
- **Status**: Not implemented

## Current Implementation

### Socket Paths
- Server: `/var/lib/spire/server/server.sock`
- Agent: `/var/lib/spire/agent/agent.sock`

### Security Mitigations
1. Socket permissions: 770 (owner/group only)
2. Directory permissions: 755
3. Socket cleanup on service start
4. Services run as root (required for workload attestation)

### Files Updated
- `/etc/spire/server/server.conf` - socket_path
- `/etc/spire/agent/agent.conf` - socket_path, server_socket_path
- `spire/scripts/register-agent.sh` - server socket path
- `spire/scripts/register-services.sh` - server socket path
- Systemd units - removed namespace isolation

## Security Trade-offs

### Lost Protections
- No filesystem namespace isolation
- System directories writable (ProtectSystem=no)
- Shared /tmp access (PrivateTmp=no)
- Home directory access (ProtectHome=no)

### Recommended Compensating Controls
1. SELinux policies for SPIRE services
2. AppArmor profiles
3. File integrity monitoring on sockets
4. Regular security audits
5. Network segmentation

## Future Improvements
1. Investigate SPIRE CLI abstract socket support
2. Implement bind mount solution for better security
3. Consider using systemd socket activation
4. Explore running SPIRE in containers with proper mounts