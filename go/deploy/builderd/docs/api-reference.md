# Builderd API Reference

## Overview

Builderd provides a ConnectRPC API for multi-tenant build execution. The API follows gRPC conventions with HTTP/2 support and includes comprehensive tenant isolation and resource management.

**Base URL**: `http://localhost:8082` (development) or configured endpoint
**Protocol**: ConnectRPC over HTTP/2
**Content-Type**: `application/json` or `application/proto`

## Authentication & Headers

| Header | Required | Description |
|--------|----------|-------------|
| `X-Tenant-ID` | Yes | Unique tenant identifier for isolation |
| `X-Customer-ID` | Yes | Customer identifier for billing |
| `Authorization` | No | Bearer token (future implementation) |
| `Content-Type` | Yes | `application/json` or `application/proto` |

## Service: BuilderService

### CreateBuild

Creates a new build request with tenant isolation and resource allocation.

**Endpoint**: `POST /builder.v1.BuilderService/CreateBuild`

**Request**:
```json
{
  "config": {
    "source": {
      "docker_image": {
        "image_uri": "ghcr.io/unkeyed/unkey:f4cfee5",
        "registry_auth": {
          "username": "optional",
          "password": "optional"
        }
      }
    },
    "target": {
      "microvm_rootfs": {
        "format": "ext4",
        "init_strategy": "INIT_STRATEGY_DIRECT_EXEC"
      }
    },
    "strategy": {
      "docker_strategy": {
        "optimize_for_size": true,
        "remove_dev_packages": true,
        "preserve_layers": false
      }
    },
    "tenant": {
      "tenant_id": "tenant-123",
      "customer_id": "customer-456",
      "tier": "TENANT_TIER_PRO"
    }
  }
}
```

**Response**:
```json
{
  "build_id": "build_01HZQQR7X8N9PN0Z2KG5WV7H3M",
  "status": "BUILD_STATUS_QUEUED",
  "created_at": "2024-01-15T10:30:00Z",
  "tenant_context": {
    "tenant_id": "tenant-123",
    "tier": "TENANT_TIER_PRO",
    "resource_limits": {
      "max_memory_bytes": 2147483648,
      "max_cpu_cores": 2,
      "timeout_seconds": 900
    }
  }
}
```

### GetBuildStatus

Retrieves the current status and progress of a build.

**Endpoint**: `GET /builder.v1.BuilderService/GetBuildStatus/{build_id}`

**Response**:
```json
{
  "build_id": "build_01HZQQR7X8N9PN0Z2KG5WV7H3M",
  "status": "BUILD_STATUS_RUNNING",
  "progress": {
    "stage": "EXTRACTION",
    "percentage": 45,
    "current_step": "Extracting Docker layers",
    "estimated_duration": "120s"
  },
  "resource_usage": {
    "memory_used_bytes": 536870912,
    "cpu_usage_percent": 75,
    "disk_used_bytes": 1073741824
  },
  "started_at": "2024-01-15T10:30:05Z",
  "updated_at": "2024-01-15T10:32:15Z"
}
```

### GetBuildArtifact

Downloads the completed rootfs artifact.

**Endpoint**: `GET /builder.v1.BuilderService/GetBuildArtifact/{build_id}`

**Response**: Binary rootfs data with appropriate content headers

### CancelBuild

Cancels a running or queued build.

**Endpoint**: `POST /builder.v1.BuilderService/CancelBuild`

**Request**:
```json
{
  "build_id": "build_01HZQQR7X8N9PN0Z2KG5WV7H3M",
  "reason": "User cancellation"
}
```

### ListBuilds

Lists builds for a tenant with filtering and pagination.

**Endpoint**: `GET /builder.v1.BuilderService/ListBuilds`

**Query Parameters**:
- `tenant_id`: Filter by tenant
- `status`: Filter by build status
- `limit`: Number of results (default: 50, max: 1000)
- `offset`: Pagination offset

**Response**:
```json
{
  "builds": [
    {
      "build_id": "build_01HZQQR7X8N9PN0Z2KG5WV7H3M",
      "status": "BUILD_STATUS_COMPLETED",
      "created_at": "2024-01-15T10:30:00Z",
      "completed_at": "2024-01-15T10:35:30Z",
      "config": { /* build config */ }
    }
  ],
  "total_count": 1,
  "has_more": false
}
```

## Build Sources

### Docker Image Source

Extract rootfs from Docker images with registry authentication support.

```json
{
  "docker_image": {
    "image_uri": "registry.example.com/namespace/image:tag",
    "registry_auth": {
      "username": "registry_user",
      "password": "registry_token"
    },
    "pull_policy": "PULL_POLICY_ALWAYS"
  }
}
```

**Supported Registries** (by tier):
- **Free**: docker.io, ghcr.io
- **Pro+**: All registries with authentication

### Git Repository Source (Planned)

```json
{
  "git_repository": {
    "clone_url": "https://github.com/owner/repo.git",
    "ref": "main",
    "auth": {
      "token": "github_token"
    },
    "dockerfile_path": "Dockerfile"
  }
}
```

### Archive Source (Planned)

```json
{
  "archive": {
    "download_url": "https://example.com/source.tar.gz",
    "format": "tar.gz",
    "auth_headers": {
      "Authorization": "Bearer token"
    }
  }
}
```

## Build Targets

### MicroVM Rootfs

```json
{
  "microvm_rootfs": {
    "format": "ext4",
    "init_strategy": "INIT_STRATEGY_DIRECT_EXEC",
    "compression": "gzip",
    "size_limit_bytes": 1073741824
  }
}
```

**Init Strategies**:
- `INIT_STRATEGY_SYSTEMD`: Use systemd as init
- `INIT_STRATEGY_DIRECT_EXEC`: Direct execution of application
- `INIT_STRATEGY_CUSTOM`: Custom init script

## Build Strategies

### Docker Strategy

```json
{
  "docker_strategy": {
    "optimize_for_size": true,
    "remove_dev_packages": true,
    "preserve_layers": false,
    "custom_commands": [
      "apt-get clean",
      "rm -rf /var/lib/apt/lists/*"
    ]
  }
}
```

## Tenant Context

### Tenant Tiers

| Tier | Enum Value | Concurrent Builds | Daily Builds | Memory | CPU |
|------|------------|-------------------|--------------|--------|-----|
| Free | `TENANT_TIER_FREE` | 1 | 5 | 512MB | 1 core |
| Pro | `TENANT_TIER_PRO` | 3 | 100 | 2GB | 2 cores |
| Enterprise | `TENANT_TIER_ENTERPRISE` | 10 | 1000 | 8GB | 4 cores |
| Dedicated | `TENANT_TIER_DEDICATED` | 50 | 10000 | 32GB | 16 cores |

### Resource Quotas

Each tenant tier includes:
- **Build Limits**: Concurrent and daily build quotas
- **Resource Limits**: Memory, CPU, disk, and network per build
- **Storage Limits**: Total artifact storage
- **Network Policies**: Registry and git host allowlists

## Status Codes

### Build Status

| Status | Description |
|--------|-------------|
| `BUILD_STATUS_UNKNOWN` | Unknown status |
| `BUILD_STATUS_QUEUED` | Waiting for execution |
| `BUILD_STATUS_RUNNING` | Currently executing |
| `BUILD_STATUS_COMPLETED` | Successfully completed |
| `BUILD_STATUS_FAILED` | Build failed |
| `BUILD_STATUS_CANCELLED` | User or system cancelled |
| `BUILD_STATUS_TIMEOUT` | Exceeded time limit |

### HTTP Status Codes

| Code | Description |
|------|-------------|
| 200 | Success |
| 400 | Invalid request parameters |
| 401 | Authentication required |
| 403 | Quota exceeded or access denied |
| 404 | Build or resource not found |
| 409 | Conflict (e.g., duplicate build) |
| 429 | Rate limit exceeded |
| 500 | Internal server error |
| 503 | Service unavailable |

## Error Handling

### Standard Error Response

```json
{
  "error": {
    "code": "QUOTA_EXCEEDED",
    "message": "Daily build limit exceeded: 5/5",
    "details": {
      "tenant_id": "tenant-123",
      "quota_type": "daily_builds",
      "current": 5,
      "limit": 5,
      "reset_time": "2024-01-16T00:00:00Z"
    }
  }
}
```

### Common Error Codes

| Code | Description |
|------|-------------|
| `QUOTA_EXCEEDED` | Resource quota violation |
| `INVALID_SOURCE` | Invalid or inaccessible source |
| `BUILD_TIMEOUT` | Build exceeded time limit |
| `STORAGE_FULL` | Insufficient storage space |
| `NETWORK_DENIED` | Network access policy violation |
| `AUTHENTICATION_FAILED` | Registry or git authentication failed |

## Rate Limiting

Rate limits are enforced per tenant:

| Tier | Requests/minute | Burst |
|------|-----------------|-------|
| Free | 60 | 10 |
| Pro | 300 | 50 |
| Enterprise | 1000 | 100 |
| Dedicated | 5000 | 500 |

Rate limit headers are included in responses:
- `X-RateLimit-Limit`: Requests per window
- `X-RateLimit-Remaining`: Remaining requests
- `X-RateLimit-Reset`: Window reset time

## Examples

### Complete Build Workflow

```bash
# 1. Create build
curl -X POST http://localhost:8082/builder.v1.BuilderService/CreateBuild \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: tenant-123" \
  -H "X-Customer-ID: customer-456" \
  -d '{
    "config": {
      "source": {
        "docker_image": {
          "image_uri": "ghcr.io/unkeyed/unkey:f4cfee5"
        }
      },
      "target": {
        "microvm_rootfs": {
          "format": "ext4",
          "init_strategy": "INIT_STRATEGY_DIRECT_EXEC"
        }
      },
      "tenant": {
        "tenant_id": "tenant-123",
        "customer_id": "customer-456",
        "tier": "TENANT_TIER_PRO"
      }
    }
  }'

# 2. Monitor progress
BUILD_ID="build_01HZQQR7X8N9PN0Z2KG5WV7H3M"
curl http://localhost:8082/builder.v1.BuilderService/GetBuildStatus/$BUILD_ID \
  -H "X-Tenant-ID: tenant-123"

# 3. Download artifact (when completed)
curl http://localhost:8082/builder.v1.BuilderService/GetBuildArtifact/$BUILD_ID \
  -H "X-Tenant-ID: tenant-123" \
  -o rootfs.ext4
```

### Monitoring and Observability

The API includes comprehensive observability:

- **Metrics**: Exposed at `/metrics` (Prometheus format)
- **Health**: Available at `/health`
- **Stats**: Service statistics at `/stats`
- **Tracing**: OpenTelemetry traces for all operations
- **Logging**: Structured JSON logs with tenant context

### Performance Considerations

- **Concurrent Builds**: Limited by tenant tier
- **Build Caching**: Docker layer caching per tenant
- **Artifact Storage**: Configurable retention policies
- **Network**: Registry access controlled by tenant tier
- **Timeouts**: Enforced per tenant tier limits

For production deployments, see the [Deployment & Operations Guide](deployment-operations-guide.md).