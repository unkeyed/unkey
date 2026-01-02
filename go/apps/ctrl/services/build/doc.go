// Package build provides container image building services.
//
// This package implements multiple build backends for container
// image creation and storage. It supports both cloud-native
// builds through Depot.dev and local builds through Docker daemon.
//
// # Architecture
//
// The package provides a unified BuildService interface with
// multiple backend implementations:
//
//   - Depot Backend: Cloud-native builds with optimized caching
//   - Docker Backend: Local builds with direct Docker integration
//
// Each backend implements:
//   - Container image creation from source code
//   - Registry pushing and management
//   - Build artifact storage in S3-compatible systems
//   - Build progress tracking and status reporting
//
// # Key Components
//
// [Depot Backend]: Integration with depot.dev for cloud builds
//
//	[Docker Backend]: Local Docker daemon integration
//	[Storage]: S3-compatible storage abstraction for build artifacts
//
// # Configuration
//
// Backends are configured through:
//   - Build platform specifications (linux/amd64, linux/arm64)
//   - Registry credentials for image pushing
//   - S3 storage configuration for build artifacts
//   - Platform-specific settings (Depot project region, Docker host access)
//
// # Usage
//
// Creating build service:
//
//	switch cfg.BuildBackend {
//	case ctrl.BuildBackendDepot:
//		buildService = depot.New(depot.Config{
//			// Depot-specific configuration
//		})
//	case ctrl.BuildBackendDocker:
//		buildService = docker.New(docker.Config{
//			// Docker-specific configuration
//		})
//	}
//
// The service provides consistent interface regardless of backend
// selection, enabling seamless switching between build systems.
//
// # Error Handling
//
// All backends provide comprehensive error handling with proper
// gRPC error codes for client communication and detailed
// logging of build failures and progress.
package build
