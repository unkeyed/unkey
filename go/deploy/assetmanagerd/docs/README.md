# AssetManagerd Documentation

Welcome to the AssetManagerd documentation. This service provides centralized asset management for virtual machine resources including kernels, rootfs images, initrd, and disk images with lifecycle management capabilities.

## Documentation Navigation

### [API Documentation](api/README.md)
Complete reference for all AssetManagerd RPCs, including:
- Service endpoints and methods
- Request/response schemas  
- Error handling patterns
- Integration examples
- Asset lifecycle management

### [Architecture Guide](architecture/README.md)
Deep dive into the service design:
- System architecture and components
- Storage backend design
- Database schema and registry
- Service interaction patterns
- Garbage collection algorithm
- Asset preparation strategies

### [Operations Manual](operations/README.md)
Production deployment and management:
- Installation and configuration
- Monitoring and metrics
- Health checks and alerting
- Storage backend configuration
- Performance tuning
- Troubleshooting guide

### [Development Setup](development/README.md)
Getting started with development:
- Build instructions
- Local development environment
- Testing strategies
- Debugging techniques
- Contributing guidelines

## Quick Links

- [Service Overview](../) - Main README with key features
- [API Proto Definition](../proto/asset/v1/asset.proto) - Protocol buffer definitions
- [QUESTIONS.md](../QUESTIONS.md) - Common questions and answers

## Service Role

AssetManagerd is one of the four pillar services in Unkey Deploy, responsible for:
- **Asset Storage** - Centralized storage for VM images and resources
- **Lifecycle Management** - Reference counting and garbage collection
- **Asset Distribution** - Efficient deployment to VM jailer environments
- **Metadata Registry** - Searchable registry with label-based discovery

## Integration Points

- **builderd** - Uploads assets to storage and registers them via API
- **metald** - Acquires and prepares assets for VM provisioning
- **Storage Backends** - Pluggable storage (local, S3, NFS, HTTP)

## Getting Help

- Check the [Operations Manual](operations/README.md) for common issues
- Review [QUESTIONS.md](../QUESTIONS.md) for design rationale
- Consult the [Architecture Guide](architecture/README.md) for implementation details