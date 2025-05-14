# Test Scenarios for v2_identities_create_identity

This document outlines test scenarios for the API endpoint that creates a new identity in the system.

## Happy Path Scenarios

- [ ] Create a basic identity with a valid externalId
- [ ] Create identity with metadata
- [ ] Create identity with rate limits
- [ ] Create identity with both metadata and rate limits
- [ ] Create multiple identities with different externalIds
- [ ] Verify identityId is returned in the response
- [ ] Create identity with complex nested metadata
- [ ] Create identity with multiple rate limits with different configurations

## Error Cases

- [ ] Attempt to create identity with empty externalId
- [ ] Attempt to create identity with externalId shorter than 3 characters
- [ ] Attempt to create identity with duplicate externalId (should return 409 CONFLICT)
- [ ] Attempt to create identity with metadata that exceeds size limit (64KB)
- [ ] Attempt to create identity with invalid rate limit configuration:
  - [ ] Negative limit value
  - [ ] Zero limit value
  - [ ] Duration less than 1000ms
  - [ ] Missing rate limit name
- [ ] Attempt to create identity with invalid JSON in request
- [ ] Attempt to create identity with missing required fields

## Security Tests

- [ ] Attempt to create identity without authentication
- [ ] Attempt to create identity with invalid authentication
- [ ] Attempt to create identity with expired token
- [ ] Attempt to create identity with insufficient permissions
- [ ] Verify correct permissions allow identity creation:
  - [ ] Test with wildcard permission ("*")
  - [ ] Test with specific permission ("identity.*.create_identity")
  - [ ] Test with multiple permissions including the required one

## Database Verification

- [ ] Verify identity record is correctly inserted in the database
- [ ] Verify identity has the correct workspace ID
- [ ] Verify identity has the correct externalId
- [ ] Verify metadata is stored correctly and retrievable
- [ ] Verify rate limits are stored correctly with proper association to the identity
- [ ] Verify audit logs are created for:
  - [ ] Identity creation
  - [ ] Rate limit creation (if applicable)
- [ ] Verify environment is set correctly

## Edge Cases

- [ ] Create identity with externalId at exactly minimum length (3 characters)
- [ ] Create identity with externalId containing special characters
- [ ] Create identity with metadata approaching size limit
- [ ] Create identity with a large number of rate limits
- [ ] Create identity with Unicode characters in externalId
- [ ] Create identity with various metadata types (arrays, nested objects, numbers, booleans)

## Performance Tests

- [ ] Measure response time for identity creation
- [ ] Test creating multiple identities in parallel
- [ ] Test system behavior under load with large metadata payloads

## Integration Tests

- [ ] Verify ability to use the created identity in downstream operations
- [ ] Verify rate limits function correctly when using the identity
- [ ] Verify metadata is correctly returned in related API calls