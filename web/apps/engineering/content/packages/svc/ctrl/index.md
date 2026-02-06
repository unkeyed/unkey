---
title: ctrl
description: "is the root package for the Unkey control plane service"
---

Package ctrl is the root package for the Unkey control plane service.

This package serves as an organizational namespace and does not contain executable code. The control plane implementation is split across several subpackages, each responsible for a distinct concern.

### Subpackages

The api subpackage provides the HTTP/2 Connect server that exposes the control plane's public API. It handles authentication, request routing, and serves as the external interface for clients interacting with the control plane.

The worker subpackage implements the Restate workflow engine integration. It hosts long-running asynchronous operations including deployment orchestration, TLS certificate lifecycle management via ACME, and container image builds. The worker registers itself with the Restate admin API for service discovery.

The services subpackage contains domain-specific service implementations that are called by both the API and worker layers. These include the deployment service for managing application deployments, the ACME service for certificate challenges, and the cluster service for node metadata.

The db subpackage provides the database access layer, including schema definitions and query functions for persistent storage operations.

The middleware subpackage contains HTTP middleware components used by the API layer, including authentication and request logging.

The pkg subpackage holds shared utilities used across the control plane, including the build backend abstraction that supports both Depot and Docker for container image building.

The workflows subpackage defines Restate workflow definitions that orchestrate multi-step operations with durable execution guarantees.

The proto subpackage contains Protocol Buffer definitions and generated code for the control plane's gRPC and Connect interfaces.

The integration subpackage provides integration test infrastructure for validating the control plane's behavior against real dependencies.
