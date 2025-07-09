# Systemd Integration for Metald

This directory contains systemd service files and deployment scripts for metald.

## Files

- `metald.service` - Production-ready systemd service unit with security hardening
- `fedora-installation.md` - Complete installation guide for Fedora 42 systems
- `metald.env.example` - Example environment configuration file
- `install.sh` - Automated installation script for systemd-based systems

## Quick Installation

```bash
# From the metald root directory
make service-install
```

## Manual Installation

```bash
# Copy service file
sudo cp contrib/systemd/metald.service /etc/systemd/system/

# Copy environment file
sudo mkdir -p /etc/metald
sudo cp contrib/systemd/metald.env.example /etc/metald/metald.env

# Edit configuration as needed
sudo vim /etc/metald/metald.env

# Install and start service
sudo systemctl daemon-reload
sudo systemctl enable metald
sudo systemctl start metald
```

## Service Management

```bash
# Check status
sudo systemctl status metald

# View logs
sudo journalctl -u metald -f

# Restart service
sudo systemctl restart metald

# Stop service
sudo systemctl stop metald
```

## Security Features

The systemd service includes comprehensive security hardening:

- Process isolation with dedicated user account
- Filesystem protection and read-only system directories
- Network and namespace restrictions
- System call filtering
- Resource limits (memory, CPU, file descriptors)
- Privilege dropping and capability restrictions

## Configuration

The service supports configuration via:

1. Environment variables in `/etc/metald/metald.env`
2. Command-line arguments (modify `ExecStart` in service file)
3. Configuration files (if implemented in metald)

## Troubleshooting

See `fedora-installation.md` for detailed troubleshooting steps and common issues.

For systemd-specific issues:

```bash
# Check service validation
sudo systemd-analyze verify /etc/systemd/system/metald.service

# Check security settings
sudo systemd-analyze security metald

# Debug service startup
sudo systemctl show metald
```