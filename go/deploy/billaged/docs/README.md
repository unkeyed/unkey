# Billaged Documentation

Welcome to the billaged service documentation. Billaged is the VM usage billing aggregation service that collects metrics from metald instances and provides real-time usage summaries for the Unkey Deploy platform.

## Documentation Index

### [Service Overview](../README.md)
Quick start guide and high-level service description.

### [API Documentation](./api/README.md)
Complete API reference with request/response examples for all RPC methods.

### [Architecture & Dependencies](./architecture/README.md)
System design, service interactions, data flow, and integration patterns.

### [Operations Guide](./operations/README.md)
Production deployment, monitoring, metrics, health checks, and troubleshooting.

### [Development Setup](./development/README.md)
Build instructions, testing, local development environment, and contribution guidelines.

## Quick Navigation

- **Getting Started**: See the [main README](../README.md) for quick start instructions
- **API Integration**: Check the [API documentation](./api/README.md) for RPC methods
- **System Design**: Review [architecture docs](./architecture/README.md) for data flow
- **Production Deploy**: Follow the [operations guide](./operations/README.md)
- **Contributing**: Read the [development guide](./development/README.md)

## Service Highlights

- **Real-time Aggregation**: Processes VM metrics with configurable intervals
- **Stateless Design**: All data stored in-memory, no database dependencies
- **High Performance**: Optimized for high-frequency metric ingestion
- **Observable**: Rich metrics, tracing, and structured logging
- **Secure**: SPIFFE/mTLS support for service authentication

## Version

Current version: **v0.1.0**