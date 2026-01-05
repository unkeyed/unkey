// Package depot provides Depot.dev build backend integration.
//
// This package implements cloud-native container builds through
// the Depot.dev platform. It provides optimized builds with
// automatic caching, parallel execution, and direct registry
// integration for production workflows.
//
// # Architecture
//
// The Depot backend integrates with:
//   - Depot.dev API for build orchestration
//   - ClickHouse for build telemetry and analytics
//   - S3 storage for build artifact persistence
//   - Registry integration for container image management
//
// # Key Features
//
// - Cloud-native builds with automatic scaling
// - Build caching for faster repeated builds
// - Parallel build execution
// - Direct registry pushing and management
// - Build artifact storage and sharing
// - Real-time build progress tracking
// - Integration with unkey platform deployment workflows
//
// # Usage
//
// Creating Depot build backend:
//
//	depotBackend := depot.New(depot.Config{
//		InstanceID:     "build-instance-001",
//		DB:             database,
//		RegistryConfig:  depot.RegistryConfig{
//			URL:      "https://registry.depot.dev",
//			Username: "x-token",
//			Password: "depot-api-token",
//		},
//		BuildPlatform:  depot.BuildPlatform{
//			Platform:     "linux/amd64",
//			Architecture: "amd64",
//		},
//		DepotConfig: depot.DepotConfig{
//			APIUrl:        "https://api.depot.dev",
//			ProjectRegion:  "us-east-1",
//		},
//		Clickhouse:     clickhouseClient,
//		Storage:        buildStorage,
//		Logger:         logger,
//	})
//
// # Build Operations
//
// The backend implements standard BuildService interface methods:
//   - CreateBuild: Start new container build
//   - GenerateUploadUrl: Generate pre-signed URLs for build artifacts
//   - GetBuild: Get build status and metadata
//   - GetBuildLogs: Stream real-time build logs
//
// # Error Handling
//
// Provides comprehensive error handling with proper HTTP status
// codes for API communication failures and build errors.
package depot
