# Test Scenarios for v2_identities_get_identity

This document outlines test scenarios for the API endpoint that retrieves identity details.

## Happy Path Scenarios

- [ ] Successfully retrieve an existing identity by ID
- [ ] Successfully retrieve an existing identity by externalId
- [ ] Retrieve identity with metadata
- [ ] Retrieve identity with rate limits
- [ ] Retrieve identity with associated keys
- [ ] Verify response structure matches specification
- [ ] Verify all expected fields are returned (ID, externalId, meta, etc.)
- [ ] Verify correct environment field is returned

## Error Cases

- [ ] Attempt to retrieve non-existent identity ID
- [ ] Attempt to retrieve non-existent externalId
- [ ] Attempt to retrieve identity with invalid ID format
- [ ] Attempt to retrieve identity with empty ID/externalId
- [ ] Attempt to retrieve deleted identity (if soft delete is used)
- [ ] Attempt to retrieve identity with malformed request
- [ ] Attempt to retrieve with both ID and externalId (if only one should be used)

## Security Tests

- [ ] Attempt to retrieve identity without authentication
- [ ] Attempt to retrieve identity with invalid authentication
- [ ] Attempt to retrieve identity with expired token
- [ ] Attempt to retrieve identity with insufficient permissions
- [ ] Attempt to retrieve identity from another workspace (should be forbidden)
- [ ] Verify correct permissions allow identity retrieval:
  - [ ] Test with wildcard permission ("*")
  - [ ] Test with specific permission ("identity.*.get_identity")
  - [ ] Test with multiple permissions including the required one

## Database Verification

- [ ] Verify returned data matches database record
- [ ] Verify metadata is correctly returned
- [ ] Verify rate limits are correctly included
- [ ] Verify timestamps are correctly formatted
- [ ] Verify relationships are correctly represented (if applicable)

## Edge Cases

- [ ] Retrieve identity with very large metadata
- [ ] Retrieve identity with many rate limits
- [ ] Retrieve identity with special characters in externalId
- [ ] Retrieve identity with Unicode characters in fields
- [ ] Retrieve recently created identity
- [ ] Retrieve identity that was recently updated

## Performance Tests

- [ ] Measure response time for identity retrieval
- [ ] Test concurrent retrieval of same identity
- [ ] Test retrieval under system load
- [ ] Compare performance with cached vs non-cached responses (if caching is implemented)
- [ ] Test performance with large metadata payload

## Integration Tests

- [ ] Verify newly created identity can be retrieved immediately
- [ ] Verify changes to identity are reflected in subsequent retrievals
- [ ] Verify consistency between list identities and get identity endpoints
- [ ] Verify audit logging works correctly for identity retrieval
- [ ] Verify rate limit information is consistent with rate limit-specific endpoints