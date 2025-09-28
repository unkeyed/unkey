// Package krane provides container orchestration with pluggable backends.
//
// Krane abstracts Docker and Kubernetes behind a unified gRPC API, enabling
// the same deployment logic to work in local development (Docker) and
// production (Kubernetes) environments.
//
// # Backends
//
// Docker: Direct container management via Docker Engine API. Suitable for
// development and single-node deployments.
//
// Kubernetes: StatefulSet and Service management via client-go. Designed
// for production clusters with stable DNS names and automatic eviction.
//
// # Usage
//
//	cfg := krane.Config{
//		Backend:     krane.Kubernetes,
//		HttpPort:    7070,
//		OtelEnabled: true,
//	}
//	err := krane.Run(context.Background(), cfg)
//
// # Service Discovery
//
//   - Docker: Dynamic port mapping on host.docker.internal
//   - Kubernetes: StatefulSet DNS names (pod.service.namespace.svc.cluster.local)
package krane
