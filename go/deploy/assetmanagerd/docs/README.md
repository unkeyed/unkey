# Assetmanagerd Documentation

Welcome to the Assetmanagerd documentation. This service manages VM assets (kernels, rootfs images) across the Unkey Deploy infrastructure with automatic build triggering and lifecycle management.

## Documentation Navigation

### [API Documentation](api/)
Complete reference for all AssetManagerService RPCs, including:
- Service endpoints and methods
- Request/response schemas with streaming support
- Automatic build triggering workflows
- Reference counting and lease management
- Error handling patterns

### [Architecture Guide](architecture/)
Deep dive into the service design:
- Asset storage backends and registry design
- Integration with builderd for automatic builds
- Reference counting and garbage collection
- SPIFFE/SPIRE mTLS authentication
- Service interaction patterns

### [Operations Manual](operations/)
Production deployment and management:
- Configuration and environment variables
- Monitoring and metrics
- Health checks and alerting
- Garbage collection and storage management
- Performance tuning

### [Development Setup](development/)
Getting started with development:
- Build instructions and testing
- Local development environment
- Storage backend configuration
- Debugging techniques

## Quick Links

- [Service Overview](../) - Main README with key features
- [API Proto Definition](../proto/asset/v1/asset.proto) - Protocol buffer definitions
- [Configuration Reference](../internal/config/config.go) - Environment variables

## Service Role

Assetmanagerd is one of the four pillar services in Unkey Deploy, responsible for:
- **Asset Storage** - Centralized management of VM kernels and rootfs images
- **Reference Counting** - Lease-based lifecycle management for garbage collection
- **Automatic Builds** - Integration with builderd to create missing assets on-demand
- **Multi-backend Storage** - Pluggable storage backends (local, S3, NFS)

## Key Features

- **Streaming Uploads** - Efficient large file handling via gRPC streaming
- **Automatic Asset Creation** - Triggers builderd when assets don't exist
- **Reference Counting** - Prevents deletion of in-use assets
- **Background GC** - Automatic cleanup of unused assets
- **Multi-tenant Support** - Tenant isolation and authentication
- **Storage Flexibility** - Multiple backend storage options

## Integration Points

- **[builderd](../../builderd/docs/)** - Automatically triggers builds for missing rootfs assets
- **metald** - // AIDEV: metald documentation needed for complete interaction description
- **billaged** - // AIDEV: billaged documentation needed for complete interaction description
- **SPIFFE/SPIRE** - mTLS authentication for all service communications

## Getting Help

- Check the [Operations Manual](operations/) for configuration and deployment
- Review the [Architecture Guide](architecture/) for implementation details
- Consult the [API Documentation](api/) for integration guidance