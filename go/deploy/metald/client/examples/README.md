# Metald Client Configuration Examples

This directory contains example VM configurations demonstrating different use cases and scenarios.

## Configuration Files

### `minimal.json`
- **Purpose**: Lightweight VM for basic tasks
- **Resources**: 1 vCPU, 512MB RAM
- **Use Cases**: Simple services, testing, CI/CD agents

```bash
# Create minimal VM
metald-cli -config=examples/configs/minimal.json create-and-boot
```

### `web-server.json`
- **Purpose**: High-performance web server
- **Resources**: 8 vCPUs, 8GB RAM (scalable to 16 vCPUs, 32GB RAM)
- **Features**: NGINX with Docker support, separate log storage
- **Use Cases**: Production web servers, load balancers, API gateways

```bash
# Create web server VM
metald-cli -config=examples/configs/web-server.json create-and-boot web-01
```

### `database.json`
- **Purpose**: High-memory database server
- **Resources**: 8 vCPUs, 32GB RAM (scalable to 16 vCPUs, 128GB RAM)
- **Features**: PostgreSQL with separate data, log, and backup storage
- **Use Cases**: Primary databases, data warehouses, analytics engines

```bash
# Create database server
metald-cli -config=examples/configs/database.json create-and-boot db-primary
```

### `development.json`
- **Purpose**: Development environment with tools
- **Resources**: 6 vCPUs, 16GB RAM (scalable to 12 vCPUs, 64GB RAM)
- **Features**: Ubuntu with development tools, Docker, workspace storage
- **Use Cases**: Developer workspaces, build environments, testing

```bash
# Create development environment
metald-cli -config=examples/configs/development.json create-and-boot dev-env
```

## Customizing Configurations

### 1. Template-Based Approach
Start with a built-in template and customize:

```bash
# Generate base configuration
metald-cli -template=standard config-gen > my-config.json

# Edit the configuration file
vim my-config.json

# Use the custom configuration
metald-cli -config=my-config.json create-and-boot
```

### 2. Override Parameters
Use CLI flags to override specific configuration:

```bash
# Use config file but override CPU and memory
metald-cli -config=web-server.json -cpu=16 -memory=65536 create-and-boot
```

### 3. Docker Image Integration
Configure VMs for specific Docker images:

```bash
# Create VM for specific Docker image
metald-cli -docker-image=redis:alpine -template=high-memory create-and-boot redis-cache
```

## Configuration Validation

Always validate configurations before use:

```bash
# Validate configuration file
metald-cli config-validate examples/configs/web-server.json

# Output validation results as JSON
metald-cli config-validate examples/configs/database.json -json
```

## Common Configuration Patterns

### High Availability Setup
```bash
# Create multiple web servers
for i in {1..3}; do
    metald-cli -config=examples/configs/web-server.json create-and-boot web-$i
done

# Create database primary and replica
metald-cli -config=examples/configs/database.json create-and-boot db-primary
metald-cli -config=examples/configs/database.json create-and-boot db-replica
```

### Development Team Setup
```bash
# Create development environments for team
for dev in alice bob charlie; do
    metald-cli -config=examples/configs/development.json create-and-boot dev-$dev
done
```

### Microservices Deployment
```bash
# Create VMs for different services
metald-cli -docker-image=my-api:latest -template=standard create-and-boot api-service
metald-cli -docker-image=my-worker:latest -template=high-cpu create-and-boot worker-service
metald-cli -config=examples/configs/database.json create-and-boot db-service
metald-cli -config=examples/configs/web-server.json create-and-boot proxy-service
```

## Best Practices

### Resource Planning
1. **Start Small**: Begin with minimal resources and scale up
2. **Enable Hotplug**: Allow memory and CPU scaling without downtime
3. **Separate Storage**: Use dedicated storage for data, logs, and backups
4. **Monitor Usage**: Track actual resource utilization

### Security Configuration
1. **Network Isolation**: Use appropriate network modes (IPv4-only for internal services)
2. **Static IPs**: Configure static IPs for database and backend services
3. **Metadata**: Include environment and team information for auditing
4. **Console Logging**: Enable console output for debugging

### Storage Configuration
1. **Root Filesystem**: Use appropriate size for OS and applications
2. **Data Storage**: Separate data storage from OS storage
3. **Log Storage**: Dedicated storage for logs to prevent disk space issues
4. **Backup Storage**: Include backup storage for critical services

### Metadata Best Practices
Include comprehensive metadata for operations:

```json
{
  "metadata": {
    "purpose": "web-server",
    "environment": "production",
    "team": "platform",
    "service": "nginx",
    "version": "1.21.6",
    "scaling_group": "web-tier",
    "backup_enabled": "true",
    "monitoring": "enabled",
    "created_by": "deployment-system",
    "cost_center": "engineering"
  }
}
```

### Network Configuration
Choose appropriate network modes:
- **dual_stack**: Most services (IPv4 + IPv6)
- **ipv4_only**: Internal services, databases
- **ipv6_only**: IPv6-only environments

## Troubleshooting

### Configuration Validation Errors
```bash
# Check for common issues
metald-cli config-validate my-config.json

# Common problems:
# - Missing root storage device
# - Invalid CPU/memory ratios
# - Incorrect network mode specifications
# - Missing required fields
```

### Resource Constraints
```bash
# Monitor VM resource usage
metald-cli info vm-12345

# Scale resources if needed
# Edit configuration file and recreate VM
# Or use hotplug for memory scaling
```

### Storage Issues
```bash
# Verify storage paths exist
ls -la /opt/vm-assets/

# Check storage device configuration
metald-cli info vm-12345 -json | jq '.config.storage'
```