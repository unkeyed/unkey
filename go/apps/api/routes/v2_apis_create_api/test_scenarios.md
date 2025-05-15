# Test Scenarios for v2_apis_create_api

This document outlines test scenarios for the API endpoint that creates a new API in the system.

## Happy Path Scenarios

- [x] Create a basic API with a valid name
- [x] Create multiple APIs with different names
- [x] Create API with a long name
- [x] Create API with special characters in the name
- [x] Create API with UUID-like name
- [x] Verify API ID is returned in the response
- [x] Verify delete protection is set to false by default
- [x] Verify the keyring (keyAuth) is correctly created and associated with the API
- [x] Verify an audit log entry is created with the correct event type and details

## Error Cases

- [x] Attempt to create API with empty name
- [x] Attempt to create API with name shorter than 3 characters
- [x] Attempt to create API with invalid JSON in request
- [x] Attempt to create API with missing required fields
- [x] Attempt to create API with malformed request body

## Security Tests

- [x] Attempt to create API without authentication
- [x] Attempt to create API with invalid authentication
- [ ] Attempt to create API with expired token
- [x] Attempt to create API with insufficient permissions
- [x] Verify correct permissions allow API creation:
  - [ ] Test with wildcard permission ("*")
  - [x] Test with specific permission ("api.*.create_api")
  - [x] Test with multiple permissions including the required one

## Database Verification

- [x] Verify API record is correctly inserted in the database
- [x] Verify API has the correct workspace ID
- [x] Verify API has the correct auth type
- [x] Verify API has non-null created timestamp
- [x] Verify API has null deleted timestamp

## Edge Cases

- [x] Create API with name at exactly minimum length (3 characters)
- [x] Create API with name containing only numbers
- [x] Create API with name containing only special characters (if allowed)
- [x] Create API with extremely long name (test boundaries)
- [x] Create API with Unicode characters in name

## Performance Tests

> Note: Performance tests were intentionally excluded because they add limited value in their current form 
> and tend to be brittle across different environments. In a real-world project, performance testing 
> would be better implemented as separate load tests with proper benchmarking tools.

- [ ] Measure response time for API creation
- [ ] Test creating multiple APIs in parallel
- [ ] Test system behavior under load

## Integration Tests

- [ ] Verify ability to use the created API in downstream operations
- [ ] Verify consistency between API creation and retrieval operations