// Package testutil provides integration test infrastructure for the API service.
//
// This package creates a complete, isolated test environment with real dependencies
// (MySQL, Redis, ClickHouse, S3, control plane) running in Docker containers. Tests
// using this package verify end-to-end behavior rather than mocking service boundaries.
//
// # Key Types
//
// The main entry point is [Harness], which orchestrates container startup, database
// seeding, and provides access to all services. Use [NewHarness] to create one.
// [TestResponse] wraps HTTP responses with typed body parsing for assertions.
//
// # Usage
//
// Create a harness at the start of your test. The harness handles container lifecycle
// and provides methods to create test data and make HTTP requests:
//
//	func TestMyEndpoint(t *testing.T) {
//	    h := testutil.NewHarness(t)
//	    h.Register(myRoute)
//
//	    ws := h.CreateWorkspace()
//	    rootKey := h.CreateRootKey(ws.ID, "api.keys.create")
//
//	    resp := testutil.CallRoute[RequestType, ResponseType](h, myRoute, headers, req)
//	    require.Equal(t, 200, resp.Status)
//	}
//
// For deployment-related tests, use [Harness.CreateTestDeploymentSetup] to create
// a workspace, project, environment, and root key in one call.
//
// # Container Dependencies
//
// The harness starts MySQL, Redis, ClickHouse, and MinIO (S3-compatible) containers.
// These are shared across tests within a package for speed, but each test gets fresh
// database state through the seeder. Container startup is parallelized to minimize
// test latency.
package testutil
