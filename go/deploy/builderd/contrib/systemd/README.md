# Builderd Systemd Integration

This directory contains systemd service files and configuration for running builderd as a system service.

## Files

- `builderd.service`: Systemd unit file for the builderd service
- `builderd.env.example`: Example environment configuration file

## Installation

### Manual Installation

1. Copy the service file to systemd directory:
   ```bash
   sudo cp builderd.service /etc/systemd/system/
   ```

2. Create builderd user and directories:
   ```bash
   sudo useradd -r -s /bin/false -d /opt/builderd builderd
   sudo mkdir -p /opt/builderd/{scratch,rootfs,workspace,data}
   sudo chown -R builderd:builderd /opt/builderd
   ```

3. Install the builderd binary:
   ```bash
   sudo cp builderd /usr/local/bin/
   sudo chmod +x /usr/local/bin/builderd
   ```

4. Configure environment (optional):
   ```bash
   sudo cp builderd.env.example /etc/default/builderd
   sudo nano /etc/default/builderd
   ```

5. Enable and start the service:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable builderd
   sudo systemctl start builderd
   ```

### Package Installation

If installing via package manager (deb/rpm), the installation steps are handled automatically.

## Service Management

```bash
# Start the service
sudo systemctl start builderd

# Stop the service
sudo systemctl stop builderd

# Restart the service
sudo systemctl restart builderd

# Check service status
sudo systemctl status builderd

# View logs
sudo journalctl -u builderd -f

# Enable auto-start on boot
sudo systemctl enable builderd

# Disable auto-start on boot
sudo systemctl disable builderd
```

## Configuration

The service can be configured through environment variables. The following methods are supported:

1. **System environment file**: `/etc/default/builderd`
2. **Service-specific environment**: Modify the `Environment=` lines in the service file
3. **Runtime environment**: Set environment variables in the shell before starting

### Key Configuration Options

- `UNKEY_BUILDERD_PORT`: Service port (default: 8082)
- `UNKEY_BUILDERD_MAX_CONCURRENT_BUILDS`: Maximum concurrent builds (default: 5)
- `UNKEY_BUILDERD_OTEL_ENABLED`: Enable OpenTelemetry (default: true in systemd, false in development)
- `UNKEY_BUILDERD_STORAGE_BACKEND`: Storage backend (local, s3, gcs)

## Directory Structure

The service expects the following directory structure:

```
/opt/builderd/
├── scratch/        # Temporary build workspaces
├── rootfs/         # Output rootfs images
├── workspace/      # Build workspace directories
└── data/           # Database and persistent data
```

## Security

The service runs as the `builderd` user with limited privileges:

- Non-login user account
- Home directory: `/opt/builderd`
- No shell access
- Resource limits configured via systemd

## Monitoring

The service provides several monitoring endpoints:

- `/health`: Health check endpoint
- `/stats`: Service statistics (JSON)
- `/metrics`: Prometheus metrics (if enabled)

## Troubleshooting

### Service won't start

1. Check service status: `sudo systemctl status builderd`
2. Check logs: `sudo journalctl -u builderd -n 50`
3. Verify binary exists: `ls -la /usr/local/bin/builderd`
4. Check permissions: `ls -la /opt/builderd`

### Permission issues

```bash
# Fix ownership
sudo chown -R builderd:builderd /opt/builderd

# Fix permissions
sudo chmod 755 /opt/builderd
sudo chmod 755 /opt/builderd/{scratch,rootfs,workspace,data}
```

### Port conflicts

Check if another service is using port 8082:
```bash
sudo netstat -tlnp | grep 8082
```

Change the port in the service file if needed:
```bash
sudo systemctl edit builderd
```

Add:
```ini
[Service]
Environment=UNKEY_BUILDERD_PORT=8083
```
