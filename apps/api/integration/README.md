# Integration Testing Strategy

This directory will contain integration tests for the Go API implementation. These tests will focus on complete workflows rather than individual endpoint behavior.

## Planned Structure

```
integration/
├── flows/           # Test flows organized by feature/domain
│   ├── apis/        # API-related workflows 
│   ├── keys/        # Key-related workflows
│   ├── identities/  # Identity-related workflows
│   └── permissions/ # Permission-related workflows
├── fixtures/        # Shared test data
├── helpers/         # Shared test utilities
└── setup/           # Test environment setup
```

## Implementation TODO

- [ ] Create a shared `TestEnvironment` that sets up databases, services, and HTTP server
- [ ] Build a `TestClient` wrapper for making HTTP requests to test endpoints
- [ ] Implement test suites using testify/suite for consistent setup/teardown
- [ ] Add build tags to control when integration tests run (`//go:build integration`)
- [ ] Create Makefile targets for running integration tests separately from unit tests
- [ ] Configure CI pipeline to run integration tests on main branch or when explicitly requested

## Design Principles

1. Tests should focus on complete user workflows rather than individual endpoints
2. Use real HTTP requests to test the entire stack
3. Organize tests by domain/feature rather than by endpoint
4. Keep test data and setup centralized to avoid duplication
5. Make tests independent and avoid inter-test dependencies

## Example Test Flow

A typical integration test should validate complete user journeys such as:

1. Create an API
2. Create a key for that API
3. Verify the key works
4. Delete the API
5. Verify the key no longer works

This validates not just individual endpoints but the entire system behavior.