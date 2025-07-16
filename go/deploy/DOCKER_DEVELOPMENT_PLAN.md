# Docker Development Environment Plan

## Overview

This document outlines the plan to create a single Docker container that runs all four Unkey deploy services (`assetmanagerd`, `billaged`, `builderd`, `metald`) for local development purposes. The container will use systemd as the process manager and implement a Docker backend for metald to create containers instead of VMs.

## Goals

1. **Production Parity**: Mirror the production deployment process as closely as possible
2. **Simplified Development**: No Firecracker/KVM setup required
3. **Container-Native**: "VMs" are actually Docker containers managed by metald
4. **Easy Debugging**: All services in one container with unified logging
5. **Fast Iteration**: Instant container startup instead of VM boot times

## Architecture

### Base Container Design

- **Base Image**: Fedora 42 (matches production environment)
- **Process Manager**: systemd (matches production)
- **Service Installation**: Use existing Makefiles and build process
- **Configuration**: Modify existing systemd services with minimal changes

### Docker Backend for metald

Replace Firecracker backend with Docker backend:

```
VM Operations → Docker Container Operations
├── CreateVM → docker create
├── BootVM → docker start
├── DeleteVM → docker rm
├── ShutdownVM → docker stop
├── PauseVM → docker pause
├── ResumeVM → docker unpause
├── RebootVM → docker restart
└── GetVMMetrics → docker stats
```

### Service Architecture

```
Single Container:
├── systemd (PID 1)
├── assetmanagerd.service (port 8083)
├── billaged.service (port 8081)
├── builderd.service (port 8082)
├── metald.service (port 8080)
└── Docker socket (mounted from host)
```

## Implementation Plan

### Phase 1: Docker Backend Implementation

1. **Create Docker Backend** (`go/deploy/metald/internal/backend/docker/`)
   - Implement `types.Backend` interface
   - Use Docker API client for container operations
   - Map VM configurations to Docker container specs
   - Handle port mapping and networking
   - Implement metrics collection via Docker stats API

2. **Asset Integration**
   - Reuse existing asset management for Docker images
   - Map VM rootfs assets to Docker images
   - Integrate with builderd for image building

3. **Configuration Updates**
   - Add Docker backend option to metald configuration
   - Update backend factory to support Docker backend

### Phase 2: Container Environment Setup

1. **Fedora-based Dockerfile**
   - Multi-stage build following LOCAL_DEPLOYMENT_GUIDE.md
   - Install development tools and dependencies
   - Build all 4 services using existing Makefiles
   - Install systemd and create service directories

2. **systemd Service Configuration**
   - Modify existing systemd service files for container environment
   - Disable TLS for all services (development only)
   - Update service endpoints from HTTPS to HTTP
   - Create billaged user and set up directories

3. **Docker Socket Integration**
   - Mount Docker socket into container
   - Configure appropriate permissions
   - Test Docker API access from within container

### Phase 3: Service Integration

1. **Environment Configuration**
   ```bash
   # Disable TLS for all services
   UNKEY_ASSETMANAGERD_TLS_MODE=disabled
   UNKEY_BILLAGED_TLS_MODE=disabled
   UNKEY_BUILDERD_TLS_MODE=disabled
   UNKEY_METALD_TLS_MODE=disabled
   
   # Configure Docker backend
   UNKEY_METALD_BACKEND=docker
   
   # Update service endpoints
   UNKEY_METALD_BILLING_ENDPOINT=http://localhost:8081
   UNKEY_METALD_ASSETMANAGER_ENDPOINT=http://localhost:8083
   UNKEY_ASSETMANAGERD_BUILDERD_ENDPOINT=http://localhost:8082
   UNKEY_BUILDERD_ASSETMANAGER_ENDPOINT=http://localhost:8083
   ```

2. **Service Dependencies**
   - Configure systemd service ordering
   - Add health checks and startup coordination
   - Handle service failures gracefully

3. **Docker Compose Integration**
   - Add deploy services to existing `deployment/docker-compose.yaml`
   - Configure networking and volume mounts
   - Set up development environment variables

### Phase 4: Testing and Validation

1. **Unit Tests**
   - Test Docker backend implementation
   - Verify VM-to-container operation mapping
   - Test asset management integration

2. **Integration Tests**
   - Test complete service startup sequence
   - Verify inter-service communication
   - Test "VM" (container) lifecycle operations

3. **End-to-End Tests**
   - Deploy sample applications as containers
   - Test port forwarding and networking
   - Verify metrics collection and billing

## Directory Structure

```
go/deploy/
├── metald/internal/backend/docker/
│   ├── client.go          # Docker backend implementation
│   ├── types.go           # Docker-specific types
│   └── metrics.go         # Docker metrics collection
├── Dockerfile.dev         # Development container
├── docker-compose.dev.yml # Development compose file
└── DOCKER_DEVELOPMENT_PLAN.md
```

## Key Implementation Details

### Docker Backend Interface Mapping

| Backend Method | Docker Operation | Notes |
|---------------|------------------|-------|
| `CreateVM()` | `docker create` | Convert VM config to container spec |
| `BootVM()` | `docker start` | Start container and configure networking |
| `DeleteVM()` | `docker rm -f` | Force remove container |
| `ShutdownVM()` | `docker stop` | Graceful shutdown with timeout |
| `PauseVM()` | `docker pause` | Pause container execution |
| `ResumeVM()` | `docker unpause` | Resume paused container |
| `RebootVM()` | `docker restart` | Restart container |
| `GetVMInfo()` | `docker inspect` | Get container state and config |
| `GetVMMetrics()` | `docker stats` | Get resource usage statistics |

### Configuration Changes

1. **TLS Disabled**: All services use `TLS_MODE=disabled`
2. **HTTP Endpoints**: Change all inter-service URLs from HTTPS to HTTP
3. **Docker Backend**: Set `UNKEY_METALD_BACKEND=docker`
4. **Socket Mount**: Mount `/var/run/docker.sock:/var/run/docker.sock`

### systemd Service Modifications

- **assetmanagerd**: Run as root, local storage backend
- **billaged**: Run as billaged user, stateless design
- **builderd**: Run as root, Docker integration enabled
- **metald**: Run as root, Docker backend enabled

## Benefits

1. **Production Parity**: Uses exact same systemd services and installation process
2. **Simplified Setup**: No Firecracker, KVM, or SPIRE setup required
3. **Familiar Tools**: Standard Docker tooling for debugging and monitoring
4. **Fast Development**: Instant container startup vs VM boot times
5. **Easy Migration**: Can switch back to separate containers easily

## Usage

### Development Workflow

```bash
# Build development container
make docker-dev-build

# Start all services
make docker-dev-up

# Create and boot a "VM" (actually a container)
docker exec -it unkey-deploy-dev metald-cli create-and-boot

# View logs
docker logs -f unkey-deploy-dev
journalctl -f  # inside container

# Stop services
make docker-dev-down
```

### Integration with Existing Docker Compose

The development container will integrate with the existing `deployment/docker-compose.yaml` to provide a complete development environment alongside the web services.

## Timeline

- **Phase 1**: Docker backend implementation (2-3 days)
- **Phase 2**: Container environment setup (1-2 days)
- **Phase 3**: Service integration (1-2 days)
- **Phase 4**: Testing and validation (1-2 days)

**Total Estimated Time**: 5-9 days

## Success Criteria

1. ✅ All 4 services start successfully in single container
2. ✅ Services communicate with each other over HTTP
3. ✅ metald can create, boot, and manage Docker containers as "VMs"
4. ✅ Asset management works with Docker images
5. ✅ Port forwarding and networking function correctly
6. ✅ Metrics collection and billing operate properly
7. ✅ Integration with existing docker-compose.yaml
8. ✅ Development workflow is faster than VM-based approach

## Future Enhancements

1. **Hot Reload**: Automatic service restart on code changes
2. **Debug Mode**: Enhanced logging and debugging capabilities
3. **Performance Monitoring**: Built-in observability stack
4. **Multi-tenancy**: Support for multiple development environments
5. **CI/CD Integration**: Automated testing and deployment