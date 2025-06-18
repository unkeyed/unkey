# Metald Example Client

This example demonstrates how to communicate with metald using mTLS via SPIFFE/SPIRE for secure service-to-service communication.

## Features

- Full mTLS support with three modes:
  - `disabled`: No TLS (development only)
  - `file`: Traditional file-based certificates
  - `spiffe`: SPIFFE/SPIRE integration (production)
- Complete VM lifecycle operations (create, boot, list, info)
- Proper tenant isolation via Bearer token authentication
- Structured logging with request/response tracing
- Example VM configurations for both Firecracker and Cloud Hypervisor backends

## Authentication

The client uses Bearer token authentication. In development mode, the token format is:
```
Authorization: Bearer dev_customer_<customer_id>
```

For example, with `--customer example-customer`, the header becomes:
```
Authorization: Bearer dev_customer_example-customer
```

## Building

```bash
# From the metald directory
cd contrib/example-client
go build -o metald-client
```

## SPIFFE/SPIRE Setup

Before using SPIFFE mode, register the client with SPIRE:

```bash
# Register the client (requires sudo)
./register-with-spire.sh

# For development/testing with relaxed security
./register-with-spire.sh dev
```

See [SPIFFE_SETUP.md](SPIFFE_SETUP.md) for detailed configuration.

## Usage Examples

### Development Mode (No TLS)

```bash
# Create and boot a VM
./metald-client \
  --addr "http://localhost:8080" \
  --customer "dev-customer" \
  --tls-mode disabled

# List all VMs
./metald-client \
  --addr "http://localhost:8080" \
  --customer "dev-customer" \
  --tls-mode disabled \
  --action list
```

### File-based TLS

```bash
# Create VM with file-based mTLS
./metald-client \
  --addr "https://metald.example.com:8080" \
  --customer "prod-customer" \
  --tls-mode file \
  --tls-cert /etc/unkey/certs/client.crt \
  --tls-key /etc/unkey/certs/client.key \
  --tls-ca /etc/unkey/certs/ca.crt
```

### SPIFFE/SPIRE (Production)

```bash
# Ensure SPIRE agent is running and registered
# Default socket path: /run/spire/sockets/agent.sock

# Create and boot VM with SPIFFE
./metald-client \
  --addr "https://metald.unkey.prod:8080" \
  --customer "prod-customer" \
  --tls-mode spiffe

# Use custom SPIFFE socket
./metald-client \
  --addr "https://metald.unkey.prod:8080" \
  --customer "prod-customer" \
  --tls-mode spiffe \
  --spiffe-socket /custom/path/to/agent.sock
```

### Specific Actions

```bash
# Just create a VM (don't boot)
./metald-client --action create --vm-id my-special-vm

# Boot an existing VM
./metald-client --action boot --vm-id my-special-vm

# Get VM information
./metald-client --action info --vm-id my-special-vm

# List all VMs
./metald-client --action list
```

## Command Line Options

| Flag | Description | Default |
|------|-------------|---------|
| `--addr` | Metald server address | `https://localhost:8080` |
| `--customer` | Customer ID for tenant isolation | `example-customer` |
| `--vm-id` | VM ID (generated if empty) | (empty) |
| `--tls-mode` | TLS mode: disabled, file, or spiffe | `disabled` |
| `--tls-cert` | TLS certificate file (file mode) | (empty) |
| `--tls-key` | TLS key file (file mode) | (empty) |
| `--tls-ca` | TLS CA file (file mode) | (empty) |
| `--spiffe-socket` | SPIFFE agent socket path | `/run/spire/sockets/agent.sock` |
| `--enable-cert-caching` | Enable certificate caching | `true` |
| `--action` | Action to perform | `create-and-boot` |

## VM Configuration

The example creates a VM with:
- 2 vCPUs (expandable to 4)
- 1GB RAM (expandable to 4GB)
- Single root filesystem
- Dual-stack networking (IPv4 DHCP + IPv6 SLAAC)
- Serial console output

Modify the `createVM` function to customize the VM configuration for your needs.

## Security Notes

1. **Customer ID**: Always provide a valid customer ID for proper tenant isolation
2. **TLS Mode**: Use `spiffe` mode in production for automatic certificate rotation
3. **Certificate Validation**: The client validates server certificates based on the configured CA
4. **Network Security**: VMs are isolated by customer ID at the network level

## Troubleshooting

### SPIFFE Connection Failed

```
Failed to create TLS provider: init SPIFFE: create X509 source: ...
```

Ensure:
1. SPIRE agent is running: `systemctl status spire-agent`
2. Socket exists: `ls -la /run/spire/sockets/agent.sock`
3. Your workload is registered with SPIRE
4. Correct socket permissions

### Certificate Errors

```
Failed to create TLS provider: validate certificates: ...
```

Check:
1. Certificate files exist and are readable
2. Certificate and key match
3. CA certificate is valid
4. Certificates haven't expired

### Connection Refused

```
Failed to create VM: unavailable: connection refused
```

Verify:
1. Metald is running: `systemctl status metald`
2. Correct address and port
3. Firewall rules allow connection
4. TLS mode matches server configuration

## Integration with Your Application

```go
import (
    "github.com/unkeyed/unkey/go/deploy/pkg/tls"
    "github.com/unkeyed/unkey/go/deploy/metald/gen/vmprovisioner/v1/vmprovisionerv1connect"
)

// Create TLS provider
tlsProvider, err := tls.NewProvider(ctx, tls.Config{
    Mode: tls.ModeSPIFFE,
    SPIFFESocketPath: "/run/spire/sockets/agent.sock",
})

// Create metald client
client := vmprovisionerv1connect.NewVmServiceClient(
    tlsProvider.HTTPClient(),
    "https://metald:8080",
)

// Use client to manage VMs
resp, err := client.CreateVm(ctx, &CreateVmRequest{...})
```