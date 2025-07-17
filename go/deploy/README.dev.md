# Unkey Deploy Services - Development Environment

This directory contains a containerized development environment for all Unkey deploy services. It provides a single Docker container running all four services (`assetmanagerd`, `billaged`, `builderd`, `metald`) with systemd as the process manager.

## Quick Start

```bash
# Navigate to deployment directory
cd deployment

# Build and start all services (including deploy services)
docker-compose up -d

# Check status
docker-compose ps

# View deploy services logs
docker-compose logs -f metald-aio

# Open shell in deploy container
docker exec -it metald-aio /bin/bash

# Test services (from host)
metald-cli -tls-mode=disabled -server=http://localhost:8090 list

# Test services (from inside container)
docker exec -it metald-aio metald-cli -tls-mode=disabled -server=http://localhost:8080 list
```

## Architecture

### Single Container Design
- **Base Image**: Fedora 42 (production parity)
- **Process Manager**: systemd (production parity)
- **Services**: All 4 deploy services in one container
- **Backend**: Docker backend for metald (containers instead of VMs)

### Service Ports
- `metald`: 8090 (exposed), 8080 (internal)
- `billaged`: 8081 (internal only)
- `builderd`: 8082 (internal only)  
- `assetmanagerd`: 8083 (internal only)

### Key Features
- **TLS Disabled**: All services use `TLS_MODE=disabled` for development
- **Docker Backend**: metald creates Docker containers instead of VMs
- **Production Parity**: Uses same systemd services and build process
- **Simplified Setup**: No Firecracker/KVM/SPIRE required

## Files and Structure

```
go/deploy/
├── Dockerfile.dev              # Development container
├── docker-compose.dev.yml      # Docker Compose configuration
├── Makefile.dev               # Development commands
├── README.dev.md              # This file
└── docker/
    └── systemd/               # systemd service files
        ├── assetmanagerd.service
        ├── billaged.service
        ├── builderd.service
        └── metald.service
```

## Usage

### Starting Services

```bash
# Build container (first time or after changes)
make -f Makefile.dev build

# Start all services
make -f Makefile.dev up

# Check if services are running
make -f Makefile.dev status
```

### Working with Services

```bash
# View service logs
make -f Makefile.dev logs

# Open shell in container
make -f Makefile.dev shell

# Inside container, check systemd status
systemctl status assetmanagerd.service
systemctl status billaged.service
systemctl status builderd.service
systemctl status metald.service

# View journal logs
journalctl -f -u metald.service
```

### Testing Services

```bash
# Run basic health checks
make -f Makefile.dev test

# Create a test VM (Docker container)
make -f Makefile.dev test-vm

# Inside container, use CLI tools
assetmanagerd-cli -tls-mode=disabled -server=http://localhost:8083 list
billaged-cli -tls-mode=disabled -server=http://localhost:8081 health
builderd-cli -tls-mode=disabled -server=http://localhost:8082 health
metald-cli -tls-mode=disabled -server=http://localhost:8080 list
```

### Creating VMs (Docker Containers)

```bash
# Inside container or using docker exec
metald-cli -tls-mode=disabled -server=http://localhost:8080 \
  -docker-image=alpine:latest create-and-boot

# List VMs
metald-cli -tls-mode=disabled -server=http://localhost:8080 list

# Get VM info
metald-cli -tls-mode=disabled -server=http://localhost:8080 info <vm-id>
```

## Configuration

### Environment Variables

The container uses `/etc/default/unkey-deploy` for configuration:

```bash
# TLS disabled for development
UNKEY_ASSETMANAGERD_TLS_MODE=disabled
UNKEY_BILLAGED_TLS_MODE=disabled
UNKEY_BUILDERD_TLS_MODE=disabled
UNKEY_METALD_TLS_MODE=disabled

# Docker backend for metald
UNKEY_METALD_BACKEND=docker

# Service endpoints (HTTP)
UNKEY_METALD_BILLING_ENDPOINT=http://localhost:8081
UNKEY_METALD_ASSETMANAGER_ENDPOINT=http://localhost:8083
UNKEY_ASSETMANAGERD_BUILDERD_ENDPOINT=http://localhost:8082
UNKEY_BUILDERD_ASSETMANAGER_ENDPOINT=http://localhost:8083
```

### Docker Backend Configuration

metald uses Docker backend with these settings:
- **Docker Host**: `unix:///var/run/docker.sock`
- **Network**: `bridge`
- **Port Range**: 30000-40000
- **Container Prefix**: `unkey-vm-`

## Troubleshooting

### Services Not Starting

```bash
# Check systemd status
make -f Makefile.dev shell
systemctl status assetmanagerd.service
systemctl status billaged.service
systemctl status builderd.service
systemctl status metald.service

# Check logs
journalctl -u metald.service
journalctl -u assetmanagerd.service
```

### Docker Access Issues

```bash
# Check Docker socket access
docker exec unkey-deploy-dev ls -la /var/run/docker.sock
docker exec unkey-deploy-dev docker version

# Ensure Docker socket is mounted
docker-compose -f docker-compose.dev.yml config
```

### Port Conflicts

```bash
# Check if ports are available
netstat -tlnp | grep -E ':(8080|8081|8082|8083)'

# Use different ports if needed
docker-compose -f docker-compose.dev.yml down
# Edit docker-compose.dev.yml ports section
docker-compose -f docker-compose.dev.yml up -d
```

## Development Workflow

### Making Changes

1. Edit service code
2. Rebuild container: `make -f Makefile.dev build`
3. Restart services: `make -f Makefile.dev down up`
4. Test changes: `make -f Makefile.dev test`

### Debugging

```bash
# Enter container
make -f Makefile.dev shell

# Check service logs
journalctl -f -u metald.service

# Test individual services
assetmanagerd-cli -tls-mode=disabled -server=http://localhost:8083 list
metald-cli -tls-mode=disabled -server=http://localhost:8080 list

# Check Docker containers created by metald
docker ps --filter "label=unkey.vm.created_by=metald"
```

## Integration with Main Docker Compose

This development environment is designed to work alongside the main `deployment/docker-compose.yaml`. The services will be available at:

- **Web services**: Ports 3000, 8787, etc. (from main compose)
- **Deploy services**: Ports 8080-8083 (from this dev environment)

Both environments share the same Docker network for seamless integration.

## Benefits

1. **Production Parity**: Uses Fedora + systemd + same build process
2. **Simplified Development**: No complex VM setup required
3. **Fast Iteration**: Instant container startup vs VM boot times
4. **Familiar Tools**: Standard Docker debugging and monitoring
5. **Easy Integration**: Works with existing docker-compose setup

## Limitations

1. **Single Container**: All services in one container (less isolation)
2. **Development Only**: TLS disabled, simplified configuration
3. **Docker Dependency**: Requires Docker socket access
4. **systemd Overhead**: Requires privileged container for systemd

## Next Steps

- [ ] Add hot reload for development
- [ ] Integrate with VS Code dev containers
- [ ] Add performance monitoring
- [ ] Create CI/CD pipeline for testing