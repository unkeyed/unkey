# Systemd Integration for Billaged

This directory contains systemd service files and deployment scripts for billaged.

## Files

- `billaged.service` - Production-ready systemd service unit
- `billaged.env.example` - Example environment configuration file

## Quick Installation

```bash
# From the billaged root directory
make service-install
```

## Manual Installation

```bash
# Copy service file
sudo cp contrib/systemd/billaged.service /etc/systemd/system/

# Copy environment file (optional)
sudo mkdir -p /etc/billaged
sudo cp contrib/systemd/billaged.env.example /etc/billaged/billaged.env

# Edit configuration as needed
sudo vim /etc/billaged/billaged.env

# Install and start service
sudo systemctl daemon-reload
sudo systemctl enable billaged
sudo systemctl start billaged
```

## Service Management

```bash
# Check status
sudo systemctl status billaged

# View logs
sudo journalctl -u billaged -f

# Restart service
sudo systemctl restart billaged

# Stop service
sudo systemctl stop billaged
```

## Configuration

The service supports configuration via:

1. Environment variables in the service file
2. Command-line arguments (modify `ExecStart` in service file)
3. Optional environment file at `/etc/billaged/billaged.env`

### Key Configuration Options

- `BILLAGED_PORT` - Service port (default: 8081)
- `BILLAGED_ADDRESS` - Bind address (default: 0.0.0.0)
- `BILLAGED_AGGREGATION_INTERVAL` - How often to print usage summaries (default: 60s)

## Integration with Metald

Billaged is designed to receive VM usage data from metald instances. To enable the integration:

1. Configure metald to use ConnectRPC billing client
2. Point metald to billaged endpoint (http://localhost:8081)
3. Billaged will automatically aggregate and print usage summaries

## Endpoints

- `/billing.v1.BillingService/*` - ConnectRPC billing service endpoints
- `/health` - Health check endpoint
- `/stats` - Current statistics and active VM count

## Security

The service runs as the `billaged` system user with minimal privileges. The installation process automatically:

- Creates the `billaged` system user
- Sets up secure directories with proper ownership
- Configures resource limits

## Troubleshooting

```bash
# Check service validation
sudo systemd-analyze verify /etc/systemd/system/billaged.service

# Debug service startup
sudo systemctl show billaged

# Check logs for errors
sudo journalctl -u billaged --no-pager
```