# Test Scenarios for v2_identities_create_identity

This document outlines test scenarios for the API endpoint that creates a new identity in the system.

## Happy Path Scenarios

- [x] Create a basic identity with a valid externalId
- [x] Create identity with metadata
- [x] Create identity with rate limits
- [x] Create identity with both metadata and rate limits
- [x] Create multiple identities with different externalIds
- [x] Verify identityId is returned in the response
- [x] Create identity with complex nested metadata
- [x] Create identity with multiple rate limits with different configurations

## Error Cases

- [x] Attempt to create identity with empty externalId
- [x] Attempt to create identity with externalId shorter than 3 characters
- [x] Attempt to create identity with duplicate externalId (should return 409 CONFLICT)
- [x] Attempt to create identity with metadata that exceeds size limit (64KB)
- [x] Attempt to create identity with invalid rate limit configuration:
  - [x] Negative limit value
  - [x] Zero limit value
  - [x] Duration less than 1000ms
  - [x] Missing rate limit name
- [x] Attempt to create identity with invalid JSON in request
- [x] Attempt to create identity with missing required fields

## Security Tests

- [x] Attempt to create identity without authentication
- [x] Attempt to create identity with invalid authentication
- [ ] ~~Attempt to create identity with expired token~~ *(Not applicable - tokens don't have time-based expiration in this implementation)*
- [x] Attempt to create identity with insufficient permissions
- [x] Verify correct permissions allow identity creation:
  - [ ] Test with wildcard permission ("*")
  - [x] Test with specific permission ("identity.*.create_identity")
  - [x] Test with multiple permissions including the required one

## Database Verification

- [x] Verify identity record is correctly inserted in the database
- [x] Verify identity has the correct workspace ID
- [x] Verify identity has the correct externalId
- [x] Verify metadata is stored correctly and retrievable
- [x] Verify rate limits are stored correctly with proper association to the identity
- [x] Verify audit logs are created for:
  - [x] Identity creation
  - [x] Rate limit creation (if applicable)
- [x] Verify environment is set correctly

## Edge Cases

- [x] Create identity with externalId at exactly minimum length (3 characters)
- [x] Create identity with externalId containing special characters
- [x] Create identity with metadata approaching size limit
- [x] Create identity with a large number of rate limits
- [x] Create identity with Unicode characters in externalId
- [x] Create identity with various metadata types (arrays, nested objects, numbers, booleans)

## Performance Tests

> Note: Performance tests were intentionally excluded because they add limited value in their current form 
> and tend to be brittle across different environments. In a real-world project, performance testing 
> would be better implemented as separate load tests with proper benchmarking tools.

- [ ] Measure response time for identity creation
- [ ] Test creating multiple identities in parallel
- [ ] Test system behavior under load with large metadata payloads

## Integration Tests

- [ ] Verify ability to use the created identity in downstream operations
- [ ] Verify rate limits function correctly when using the identity
- [ ] Verify metadata is correctly returned in related API calls