// Package openapi provides OpenAPI specification and schema documentation.
//
// This service implements retrieval and analysis of OpenAPI
// specifications from deployed applications. It enables the control
// plane to understand the API surface of running deployments
// and provide comprehensive API documentation.
//
// # Architecture
//
// The service queries deployed applications to extract their
// OpenAPI specifications and makes them available through
// the control plane's API. This enables:
//   - API discovery and documentation generation
//   - Integration with external API documentation tools
//   - Validation of API specifications
//   - Schema analysis for deployment compatibility
//
// # Implementation Details
//
// The service:
//   - Connects to deployed applications via HTTP/HTTPS
//   - Retrieves OpenAPI specifications in multiple formats
//   - Validates specification structure and syntax
//   - Caches specifications to reduce repeated API calls
//   - Provides structured responses for integration
//
// # Key Features
//
// - Multi-format support: JSON, OpenAPI, Swagger
// - Version compatibility: Handles specification versioning
// - Validation: Ensures API contract compliance
// - Caching: Optimizes performance for repeated requests
// - Error handling: Comprehensive error reporting
//
// # Usage
//
// Creating OpenAPI service:
//
//	openapiSvc := openapi.New(database, logger)
//
//	// Register with Connect server
//	mux.Handle(ctrlv1connect.NewOpenApiServiceHandler(openapiSvc))
//
// # Integration
//
// This service integrates with:
//   - Deployment service: To get application endpoints
//   - Krane agents: To fetch specifications from running containers
//   - Database: For persistence and caching
//   - Control plane: For orchestration and management
package openapi
