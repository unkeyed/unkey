# Gateway Config Seeder

This utility creates gateway configurations in the database for testing the gateway service.

## Usage

```bash
# Basic usage with required flags
unkey seed-gateway \
  --hostname="api.myapp.com" \
  --deployment-id="deploy-123" \
  --vm-ids="vm-001,vm-002,vm-003" \
  --vm-ips="192.168.1.10,192.168.1.11,192.168.1.12" \
  --vm-ports="8080,8080,8080" \
  --dsn="user:password@tcp(localhost:3306)/database_name"

# Or use environment variable for DSN and default ports
export DATABASE_DSN="user:password@tcp(localhost:3306)/database_name"
unkey seed-gateway \
  --hostname="api.myapp.com" \
  --deployment-id="deploy-123" \
  --vm-ids="vm-001,vm-002,vm-003" \
  --vm-ips="192.168.1.10,192.168.1.11,192.168.1.12"

# Create disabled gateway with custom region
unkey seed-gateway \
  --hostname="staging.myapp.com" \
  --deployment-id="deploy-staging" \
  --vm-ids="vm-staging-001" \
  --vm-ips="10.0.1.100" \
  --vm-ports="3000" \
  --region="us-west-2" \
  --enabled=false

# Test the gateway
curl -H "Host: api.myapp.com" http://localhost:7070/
```

## Flags

- `--hostname` (required): The hostname/domain for the gateway (e.g., "api.myapp.com")
- `--deployment-id` (required): Unique deployment identifier 
- `--vm-ids` (required): Comma-separated list of VM IDs that serve this gateway
- `--vm-ips` (required): Comma-separated list of VM private IPs (must match vm-ids order)
- `--vm-ports` (optional): Comma-separated list of VM ports, or single port for all VMs (default: "8080")
- `--region` (optional): Region for the VMs (default: "us-east-1")
- `--enabled` (optional): Whether the gateway is enabled (default: true)
- `--dsn` (optional): Database connection string (can use DATABASE_DSN env var instead)

## What it creates

- VM entries in the `vms` table with specified IPs, ports, and status
- Gateway configuration protobuf for the specified hostname
- Associates the gateway with the provided VM IDs
- Stores the protobuf blob in the `gateways` table
- Verifies both VM and gateway configurations can be read back and unmarshaled

## Environment Variables

- `DATABASE_DSN` - Optional. MySQL connection string for the partition database (alternative to `--dsn` flag).

## Output

The script will show:
1. Progress creating VM entries with their IP addresses and ports
2. Confirmation that the gateway config was created
3. Verification that it can be read back and unmarshaled  
4. Details of the configuration (deployment ID, enabled status, VM count)
5. List of all VMs associated with the gateway, including IP:port and status
6. Curl commands to test the gateway

## Examples

```bash
# Production gateway with multiple VMs
unkey seed-gateway \
  --hostname="api.unkey.dev" \
  --deployment-id="prod-api-v1.2.3" \
  --vm-ids="vm-prod-001,vm-prod-002,vm-prod-003" \
  --vm-ips="10.0.1.10,10.0.1.11,10.0.1.12" \
  --vm-ports="8080"

# Development gateway with single VM on different port
unkey seed-gateway \
  --hostname="dev.unkey.localhost" \
  --deployment-id="dev-local" \
  --vm-ids="vm-dev-001" \
  --vm-ips="127.0.0.1" \
  --vm-ports="3000" \
  --region="local"

# Load balancer test with many VMs on different ports
unkey seed-gateway \
  --hostname="loadtest.unkey.dev" \
  --deployment-id="loadtest-setup" \
  --vm-ids="vm-lb-001,vm-lb-002,vm-lb-003,vm-lb-004,vm-lb-005" \
  --vm-ips="10.1.1.10,10.1.1.11,10.1.1.12,10.1.1.13,10.1.1.14" \
  --vm-ports="8080,8081,8082,8083,8084"
```