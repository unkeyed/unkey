// Package docker provides local Docker build backend integration.
//
// This package implements container image building through the local
// Docker daemon. It supports builds from Dockerfile paths
// and direct container image management without external dependencies.
//
// # Architecture
//
// The Docker backend provides:
//   - Local Docker daemon integration for container builds
//   - Docker image creation from source code
//   - Build artifact storage in S3-compatible systems
//   - Real-time build progress tracking
//   - Integration with unkey deployment workflows
//
// # Key Features
//
// - Local build execution without external service dependencies
//   - Docker daemon communication for container management
//   - Build from Dockerfile paths for reproducible builds
//   - Container image creation and management
//   - S3 storage for build artifact sharing
//   - Integration with deployment service for automatic updates
//
// # Usage
//
// Creating Docker build backend:
//
//	dockerBackend := docker.New(docker.Config{
//		InstanceID:     "build-instance-001",
//		DB:             database,
//		BuildPlatform:  docker.BuildPlatform{
//			Platform:     "linux/amd64",
//			Architecture: "amd64",
//		},
//		Storage:        buildStorage,
//		Logger:         logger,
//	})
//
// # Build Operations
//
// The backend implements standard BuildService interface methods:
//   - CreateBuild: Start new container build from Dockerfile
//   - GenerateUploadUrl: Generate pre-signed URLs for Docker images
//   - GetBuild: Get build status and metadata
//   - GetBuildLogs: Stream real-time build logs
//
// # Error Handling
//
// Provides comprehensive error handling with proper HTTP status
// codes for Docker daemon communication failures and build errors.
package docker
