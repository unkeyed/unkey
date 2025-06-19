# Builderd Documentation

Welcome to the Builderd documentation. This service provides multi-tenant build execution for various source types, producing optimized rootfs images for Firecracker microVM execution.

## Documentation Navigation

### [API Documentation](api/)
Complete reference for all Builderd RPCs, including:
- Service endpoints and methods
- Request/response schemas
- Error handling patterns
- Integration examples
- Multi-tenant authentication

### [Architecture Guide](architecture/)
Deep dive into the service design:
- System architecture and components
- Build execution pipeline
- Multi-tenant isolation strategies
- Storage and caching design
- Service interaction patterns

### [Operations Manual](operations/)
Production deployment and management:
- Installation and configuration
- Monitoring and metrics
- Health checks and alerting
- Resource management
- Performance tuning
- Troubleshooting guide

### [Development Setup](development/)
Getting started with development:
- Build instructions
- Local development environment
- Testing strategies
- Debugging techniques
- Contributing guidelines

## Quick Links

- [Service Overview](../) - Main README with key features
- [API Proto Definition](../proto/builder/v1/builder.proto) - Protocol buffer definitions
- [QUESTIONS.md](../QUESTIONS.md) - Common questions and answers

## Service Role

Builderd is one of the four pillar services in Unkey Deploy, responsible for:
- **Build Execution** - Transforming source images into optimized rootfs
- **Multi-tenant Isolation** - Secure, isolated build environments per tenant
- **Resource Management** - Enforcing quotas and limits per tenant tier
- **Artifact Production** - Creating rootfs images for microVM deployment

## Integration Points

- **assetmanagerd** - Registers built artifacts for centralized management
- **metald** - (Future) Will consume rootfs outputs for VM provisioning
- **billaged** - (Future) Will receive usage metrics for billing

## Getting Help

- Check the [Operations Manual](operations/) for common issues
- Review [QUESTIONS.md](../QUESTIONS.md) for design rationale
- Consult the [Architecture Guide](architecture/) for implementation details