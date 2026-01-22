// Package build provides container image building via [Depot.dev].
//
// Unkey uses Depot for container builds because it provides isolated build
// environments with automatic caching, eliminating the need to manage buildkit
// infrastructure. Each Unkey project gets a dedicated Depot project, ensuring
// cache isolation between tenants while sharing cache within a project.
//
// # Architecture
//
// The build service operates as a Restate workflow step within the deployment
// pipeline. When a deployment requires building from source, the deploy worker
// calls [Depot.BuildDockerImage] which:
//
//  1. Creates or retrieves a Depot project for the Unkey project
//  2. Acquires a build machine from Depot's infrastructure
//  3. Connects to the buildkit instance on that machine
//  4. Streams build context from S3 and executes the build
//  5. Pushes the resulting image to the configured registry
//  6. Records build step telemetry to ClickHouse
//
// # Usage
//
// Create a Depot backend and register it with Restate:
//
//	backend := build.New(build.Config{
//	    InstanceID: "build-instance-001",
//	    DB:         database,
//	    DepotConfig: build.DepotConfig{
//	        APIUrl:        "https://api.depot.dev",
//	        ProjectRegion: "us-east-1",
//	    },
//	    RegistryConfig: build.RegistryConfig{
//	        URL:      "registry.depot.dev",
//	        Username: "x-token",
//	        Password: depotToken,
//	    },
//	    BuildPlatform: build.BuildPlatform{
//	        Platform:     "linux/amd64",
//	        Architecture: "amd64",
//	    },
//	    Clickhouse: clickhouseClient,
//	    Logger:     logger,
//	})
//
// The backend implements [hydrav1.BuildServiceServer] and exposes
// [Depot.BuildDockerImage] as an RPC endpoint.
//
// # Cache Policy
//
// New Depot projects are created with a cache policy of 50GB retained for 14
// days. This balances build speed (cache hits) against storage costs.
//
// [Depot.dev]: https://depot.dev
package build
