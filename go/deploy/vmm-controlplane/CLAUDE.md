# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

VMM Control Plane (VMCP) is a ConnectRPC-based API service that provides a controlled frontend for creating VMs using Virtual Machine Monitors (VMMs). It acts as an intermediary between the Deploy control plane and VMM instances (Cloud Hypervisor, Firecracker), offering a pluggable backend architecture.

## Common Development Commands

Since the project is in early stages, build tooling needs to be established. When implementing:

```bash
# Standard Go commands (until proper tooling is added)
go mod tidy              # Update dependencies
go build ./cmd/api       # Build the API server
go test ./...            # Run all tests
go fmt ./...             # Format code
go vet ./...             # Run static analysis
```

## Architecture

The project follows a standard Go service architecture with pluggable VMM backends:

- **API Layer**: ConnectRPC/protobuf-based API with unified VM operations
- **Service Layer**: Business logic for VM lifecycle management
- **Backend Layer**: Abstracted interface for different VMM implementations
- **Client Layer**: HTTP clients for communicating with VMM instances via Unix sockets

Key architectural decisions:
- Uses buf/ConnectRPC with protobufs for API contracts
- Backend abstraction enables support for multiple VMMs (Cloud Hypervisor, Firecracker)
- Designed to be called by the Deploy control plane with controlled VM creation parameters
- Unix socket communication for security and performance

## Implementation Notes

When implementing features:

1. **API Design**: Define unified protobuf contracts that work across different VMM backends
2. **Backend Pattern**: Implement the Backend interface for each VMM type (see internal/backend/types/backend.go)
3. **Service Pattern**: Implement ConnectRPC service handlers in dedicated service packages
4. **Error Handling**: Use ConnectRPC's error model with proper status codes
5. **Configuration**: Use environment variables for VMM endpoints and backend selection

## Required Setup Tasks

The following need to be created when starting development:

1. **buf configuration**: Create buf.yaml and buf.gen.yaml for protobuf generation
2. **Proto definitions**: Define API contracts based on Cloud Hypervisor OpenAPI spec
3. **Makefile**: Add common development commands
4. **.gitignore**: Replace current Node.js gitignore with Go-specific patterns
5. **Docker support**: Add Dockerfile for containerized deployment

## **AIDEV Anchor Comment System**

**Purpose:** Maintain searchable inline documentation that persists across development sessions.

**Format:** Use `AIDEV-NOTE:`, `AIDEV-TODO:`, `AIDEV-QUESTION:`, `AIDEV-BUSINESS_RULE:` prefixes in comments.

**When to Add Anchors:**
- Code that is complex or non-obvious
- Critical business logic implementations
- Known workarounds or technical debt
- Areas prone to bugs or requiring special attention
- Backend-specific implementation details

**Workflow:**
1. **Before modifying code:** Grep for existing `AIDEV-*` anchors in relevant directories
2. **During development:** Add anchors for new complex or important code
3. **After changes:** Update existing anchors that reference modified code
4. **Session end:** Document new learnings in appropriate reference files

### **AIDEV-BUSINESS_RULE** Pattern

- Always `buf lint` after making changes to any `*.proto` file
- All VMM backends must implement the Backend interface consistently
- Configuration must support multiple VMM types via VMCP_BACKEND environment variable

### OpenTelemetry Implementation

The codebase includes comprehensive OpenTelemetry instrumentation:

### Configuration
- All OTEL config uses `UNKEY_VMCP_` prefix for environment variables
- Sampling rate is configurable from 0.0 to 1.0
- Supports both OTLP push and Prometheus pull for metrics
- Can run with OTEL completely disabled

### Key Implementation Details
1. **Schema Conflict Resolution**: Uses semconv v1.24.0 with OTEL v1.36.0 to avoid conflicting schema URLs (see AIDEV-NOTE in otel.go)
2. **Parent-based Sampling**: Honors upstream trace decisions while applying local sampling rate to root spans
3. **Error Always-On**: Errors are captured via re-sampling even when sampling rate is low
4. **Dual Metrics Export**: Both OTLP (push) and Prometheus (pull) endpoints are supported
5. **HTTP Client Instrumentation**: Works with Unix sockets for Cloud Hypervisor communication

### Testing
- Use `make o11y` to start Grafana LGTM stack (requires Podman)
- Prometheus metrics available on port 9464, not 9090 (to avoid conflicts)
- ✅ Comprehensive unit tests implemented for OpenTelemetry configuration parsing and validation in config_test.go
- AIDEV-TODO: Add integration tests for trace/metric collection with mock OTLP endpoint

# Memory Notes

- Every file MUST end with a newline.
- OpenTelemetry resource creation requires careful version matching to avoid schema conflicts
- Use Podman over Docker
- Project renamed from cloud-hypervisor-controlplane to vmm-controlplane to support multiple VMM backends
- Backend abstraction layer enables Cloud Hypervisor and Firecracker support through common interface
- Always format *.go files with goimports -w