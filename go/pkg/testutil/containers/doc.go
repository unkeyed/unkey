// Package containers provides testing utilities for the Unkey project,
// particularly focused on containerized dependencies for integration testing.
//
// It simplifies the process of setting up external dependencies like MySQL
// databases in test environments using Docker containers. This allows tests
// to run against real services without manual setup or configuration.
//
// Common use cases include:
// - Setting up a MySQL database for integration tests
// - Creating isolated test environments that can be easily torn down
// - Testing database interactions with actual database instances
package containers
