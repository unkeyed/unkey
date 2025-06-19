# AssetManagerd Documentation

Welcome to the AssetManagerd documentation. This service provides centralized asset management for VM-related resources across the Unkey Deploy infrastructure.

## Documentation Navigation

### [API Documentation](api/)
Complete reference for all AssetManagerd RPCs, including:
- Service endpoints and methods
- Request/response schemas
- Error handling patterns
- Integration examples
- Authentication requirements

### [Architecture Guide](architecture/)
Deep dive into the service design:
- System architecture and components
- Storage backend design
- Database schema and rationale
- Service interaction patterns
- Concurrency and consistency models

### [Operations Manual](operations/)
Production deployment and management:
- Installation and configuration
- Monitoring and metrics
- Health checks and alerting
- Backup and recovery procedures
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
- [API Proto Definition](../proto/asset/v1/asset.proto) - Protocol buffer definitions
- [QUESTIONS.md](../QUESTIONS.md) - Common questions and answers
- [TODO.md](../TODO.md) - Roadmap and planned features

## Service Role

AssetManagerd is one of the four pillar services in Unkey Deploy, working alongside:
- **metald** - VM lifecycle management (primary consumer)
- **builderd** - Container and VM image building (primary producer)
- **billaged** - Usage tracking and billing

## Getting Help

- Check the [Operations Manual](operations/) for common issues
- Review [QUESTIONS.md](../QUESTIONS.md) for design rationale
- Consult the [Architecture Guide](architecture/) for implementation details