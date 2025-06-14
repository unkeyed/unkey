# Jailer Permissions and Setup Guide

## Overview

Metald is responsible for spawning and managing jailer processes. This requires specific system permissions and configuration.

## Architecture

```
metald (running as metald user)
  │
  ├─> spawns jailer process (requires CAP_SYS_ADMIN)
  │     │
  │     ├─> jailer sets up chroot, namespaces
  │     ├─> jailer drops privileges to configured UID/GID
  │     └─> jailer execs firecracker inside chroot
  │
  └─> communicates with firecracker via unix socket
```

## Required Setup

### 1. Create metald User

```bash
# Create system user for metald
sudo useradd -r -s /bin/false -d /opt/metald metald
```

### 2. Configure Jailer Binary Permissions

The jailer binary must be:
- Owned by root
- Executable by metald user
- Have appropriate permissions

```bash
# Ensure jailer binary has correct ownership
sudo chown root:root /usr/bin/jailer
sudo chmod 755 /usr/bin/jailer

# Firecracker binary should also be accessible
sudo chown root:root /usr/bin/firecracker
sudo chmod 755 /usr/bin/firecracker
```

### 3. Grant Capabilities to Metald

Metald needs specific capabilities to spawn jailer processes:

#### Option A: Using systemd (Recommended)

Use the provided systemd service file which sets:
- `AmbientCapabilities=CAP_SYS_ADMIN CAP_NET_ADMIN CAP_DAC_OVERRIDE`
- These allow metald to execute jailer and manage network namespaces

```bash
sudo cp contrib/systemd/metald-jailer.service /etc/systemd/system/metald.service
sudo systemctl daemon-reload
sudo systemctl start metald
```

#### Option B: Using setcap (Development)

```bash
# Grant capabilities to metald binary
sudo setcap 'cap_sys_admin,cap_net_admin,cap_dac_override+eip' /usr/local/bin/metald
```

#### Option C: Using sudo (Testing Only)

```bash
# Add sudoers rule for metald to run jailer
echo "metald ALL=(root) NOPASSWD: /usr/bin/jailer" | sudo tee /etc/sudoers.d/metald
```

### 4. Directory Permissions

```bash
# Create and configure jailer chroot base directory
sudo mkdir -p /srv/jailer
sudo chown metald:metald /srv/jailer
sudo chmod 755 /srv/jailer

# Create metald directories
sudo mkdir -p /opt/metald/{sockets,logs,data}
sudo chown -R metald:metald /opt/metald
sudo chmod 755 /opt/metald/*

# Network namespace directory (if using netns)
sudo mkdir -p /var/run/netns
# This remains owned by root but metald needs CAP_NET_ADMIN to use it
```

## Troubleshooting

### Permission Denied Errors

If you see "permission denied" when metald tries to start jailer:

1. **Check capabilities**:
   ```bash
   getcap /usr/local/bin/metald
   # Should show: cap_sys_admin,cap_net_admin,cap_dac_override+eip
   ```

2. **Verify systemd service**:
   ```bash
   systemctl show metald | grep -E "(Ambient|Bounding)Capabilities"
   ```

3. **Check jailer binary**:
   ```bash
   ls -la /usr/bin/jailer
   # Should be: -rwxr-xr-x 1 root root
   ```

4. **Test manually**:
   ```bash
   # As metald user, try to execute jailer
   sudo -u metald /usr/bin/jailer --version
   ```

### Socket Connection Issues

If metald can't connect to the firecracker socket:

1. **Check socket path**:
   - Without jailer: `/opt/metald/sockets/fc-{id}.sock`
   - With jailer: `/srv/jailer/vm-{id}/root/tmp/firecracker.socket`

2. **Verify socket exists**:
   ```bash
   find /srv/jailer -name "*.socket" -ls
   ```

3. **Test connectivity**:
   ```bash
   # Use the debug script
   sudo ./scripts/debug-jailer-socket.sh
   ```

### SELinux/AppArmor

These security frameworks may block jailer execution:

```bash
# For SELinux
sudo setenforce 0  # Temporary disable for testing
# Or create proper SELinux policy

# For AppArmor
sudo aa-complain /usr/bin/jailer  # Set to complain mode
# Or create proper AppArmor profile
```

## Security Considerations

1. **Minimal Privileges**: Metald runs as non-root with only required capabilities
2. **Jailer Isolation**: Each VM runs in isolated chroot with dropped privileges
3. **Network Isolation**: Optional network namespaces per VM
4. **Resource Limits**: Cgroups enforce memory/CPU limits per VM

## Example Configuration

```bash
# /etc/metald/metald.env
UNKEY_METALD_BACKEND=firecracker
UNKEY_METALD_JAILER_ENABLED=true
UNKEY_METALD_JAILER_BINARY=/usr/bin/jailer
UNKEY_METALD_FIRECRACKER_BINARY=/usr/bin/firecracker
UNKEY_METALD_JAILER_UID=1000
UNKEY_METALD_JAILER_GID=1000
UNKEY_METALD_JAILER_CHROOT_DIR=/srv/jailer
UNKEY_METALD_JAILER_NETNS=true
UNKEY_METALD_JAILER_PIDNS=true
```

## Summary

The key insight is that **metald manages jailer processes**, not systemd. This requires:
1. Metald to have `CAP_SYS_ADMIN` capability
2. Jailer binary to be executable by metald
3. Proper directory permissions for chroot environments
4. Socket connectivity between metald and firecracker inside chroot