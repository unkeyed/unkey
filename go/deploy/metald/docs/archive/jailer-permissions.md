# Jailer Permissions and Setup Guide

## Overview

Metald includes an integrated jailer implementation that provides secure VM isolation. This requires specific system permissions and configuration.

## Architecture

```
metald (running with capabilities)
  │
  ├─> forks child process for VM
  │     │
  │     ├─> enters network namespace
  │     ├─> creates TAP device inside namespace
  │     ├─> sets up chroot environment
  │     ├─> drops privileges to configured UID/GID
  │     └─> execs firecracker inside chroot
  │
  └─> communicates with firecracker via unix socket
```

## Required Setup

### 1. Create metald User

```bash
# Create system user for metald
sudo useradd -r -s /bin/false -d /opt/metald metald
```

### 2. Configure Firecracker Binary Permissions

The firecracker binary must be accessible:

```bash
# Ensure firecracker binary has correct ownership and permissions
sudo chown root:root /usr/local/bin/firecracker
sudo chmod 755 /usr/local/bin/firecracker
```

### 3. Grant Capabilities to Metald

Metald needs specific capabilities for the integrated jailer to function:

```bash
# Grant required capabilities to metald binary
sudo setcap 'cap_sys_admin,cap_net_admin,cap_sys_chroot,cap_setuid,cap_setgid,cap_mknod,cap_dac_override+ep' /usr/local/bin/metald
```

### 4. Configure Jailer User/Group

Create a dedicated user for running firecracker processes:

```bash
# Create jailer user (typically UID/GID 977/976 for system services)
sudo groupadd -g 976 firecracker
sudo useradd -u 977 -g 976 -s /bin/false -d /dev/null firecracker
```

### 5. Set Up Chroot Directory

```bash
# Create and configure chroot base directory
sudo mkdir -p /srv/jailer
sudo chown root:root /srv/jailer
sudo chmod 755 /srv/jailer
```

### 6. Configure systemd Service

When running metald as a systemd service:

```ini
[Service]
# Run as metald user with required capabilities
User=metald
Group=metald

# Capabilities are preserved through ambient capabilities
AmbientCapabilities=CAP_SYS_ADMIN CAP_NET_ADMIN CAP_SYS_CHROOT CAP_SETUID CAP_SETGID CAP_MKNOD CAP_DAC_OVERRIDE

# Security restrictions
PrivateTmp=yes
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=/srv/jailer /opt/metald/workdir /dev/kvm /dev/net/tun
```

## Capability Requirements

The integrated jailer requires these capabilities:

| Capability | Purpose |
|------------|---------|
| `CAP_SYS_ADMIN` | Enter namespaces, mount operations |
| `CAP_NET_ADMIN` | Create TAP devices, configure network |
| `CAP_SYS_CHROOT` | Chroot to jail directory |
| `CAP_SETUID` | Drop to unprivileged UID |
| `CAP_SETGID` | Drop to unprivileged GID |
| `CAP_MKNOD` | Create device nodes in chroot |
| `CAP_DAC_OVERRIDE` | Access files during setup |

## Security Considerations

1. **Minimal Privileges**: Metald only needs capabilities, not root
2. **Privilege Dropping**: Each VM process drops all privileges after setup
3. **Isolation**: Each VM runs in separate chroot with dedicated UID/GID
4. **No Sudo Required**: Capabilities eliminate need for sudo or setuid binaries

## Troubleshooting

### Permission Denied Errors

```bash
# Check metald has required capabilities
getcap /usr/local/bin/metald

# Verify output shows all required capabilities
# Expected: cap_sys_admin,cap_net_admin,cap_sys_chroot,cap_setuid,cap_setgid,cap_mknod,cap_dac_override+ep
```

### TAP Device Creation Failed

```bash
# Ensure /dev/net/tun is accessible
ls -la /dev/net/tun

# Check CAP_NET_ADMIN is granted
# TAP devices require this capability
```

### Chroot Setup Failed

```bash
# Verify chroot directory permissions
ls -la /srv/jailer/

# Check disk space
df -h /srv/jailer
```

## Migration from External Jailer

If migrating from the external jailer binary:

1. Remove sudo rules for jailer binary
2. Remove setuid bit from jailer binary (no longer needed)
3. Grant capabilities to metald as shown above
4. No other changes required - same UID/GID/chroot configuration