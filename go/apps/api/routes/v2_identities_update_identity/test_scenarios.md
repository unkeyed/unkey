# Test Scenarios for v2_identities_update_identity

This document outlines test scenarios for the API endpoint that updates an existing identity.

## Happy Path Scenarios

- [ ] Successfully update identity metadata
- [ ] Successfully update identity with empty metadata (clearing existing metadata)
- [ ] Update identity by ID
- [ ] Update identity by externalId
- [ ] Update identity with new rate limits
- [ ] Update identity by removing rate limits
- [ ] Update identity by modifying existing rate limits
- [ ] Verify response structure matches specification
- [ ] Verify partial updates work correctly (only updating specified fields)
- [ ] Update identity with unchanged data (idempotent operation)

## Error Cases

- [ ] Attempt to update non-existent identity ID
- [ ] Attempt to update non-existent externalId
- [ ] Attempt to update identity with invalid ID format
- [ ] Attempt to update identity with empty ID/externalId
- [ ] Attempt to update deleted identity
- [ ] Attempt to update identity with malformed request
- [ ] Attempt to update with both ID and externalId (if only one should be used)
- [ ] Attempt to update with metadata exceeding size limits
- [ ] Attempt to update with invalid rate limit configuration:
  - [ ] Negative limit value
  - [ ] Zero limit value
  - [ ] Duration less than minimum allowed
  - [ ] Missing rate limit name

## Security Tests

- [ ] Attempt to update identity without authentication
- [ ] Attempt to update identity with invalid authentication
- [ ] Attempt to update identity with expired token
- [ ] Attempt to update identity with insufficient permissions
- [ ] Attempt to update identity from another workspace (should be forbidden)
- [ ] Verify correct permissions allow identity updates:
  - [ ] Test with wildcard permission ("*")
  - [ ] Test with specific permission ("identity.*.update_identity")
  - [ ] Test with multiple permissions including the required one

## Database Verification

- [ ] Verify identity record is correctly updated in database
- [ ] Verify old metadata is completely replaced by new metadata
- [ ] Verify rate limits are correctly updated:
  - [ ] New rate limits are created
  - [ ] Removed rate limits are deleted
  - [ ] Modified rate limits are updated
- [ ] Verify updated timestamp is correctly set
- [ ] Verify audit log entry is created for the update
- [ ] Verify workspace ID is validated during update

## Edge Cases

- [ ] Update identity with very large metadata
- [ ] Update identity with many rate limits
- [ ] Update identity with special characters in metadata
- [ ] Update identity with Unicode characters in fields
- [ ] Update recently created identity
- [ ] Update identity multiple times in succession
- [ ] Update identity with metadata approaching size limit

## Concurrency Tests

- [ ] Attempt concurrent updates of the same identity
- [ ] Test race conditions between update and other operations

## Performance Tests

- [ ] Measure response time for identity updates
- [ ] Test updates with varying sizes of metadata
- [ ] Test system performance when multiple identities are updated concurrently
- [ ] Test updates with large number of rate limits

## Integration Tests

- [ ] Verify updated identity data is immediately reflected in get identity endpoint
- [ ] Verify rate limit changes are immediately effective
- [ ] Verify updates to identity affect associated keys correctly
- [ ] Verify analytics record update events correctly
- [ ] Verify audit trail contains complete update information
- [ ] Verify webhooks trigger correctly on identity updates (if implemented)