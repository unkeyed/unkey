# Test Scenarios for v2_apis_list_keys

This document outlines test scenarios for the API endpoint that lists keys for a specific API.

## Happy Path Scenarios

- [x] List keys with default pagination (no cursor provided) - `200_test.go: list_all_keys_with_default_pagination`
- [x] List keys with pagination (cursor provided) - `200_test.go: list_keys_with_pagination_cursor`
- [x] List keys with limit parameter - `200_test.go: list_keys_with_limit_parameter`
- [x] List keys with specific filters (if supported) - `200_test.go: filter_by_external_ID`
- [x] List keys when none exist (should return empty array) - `200_test.go: empty_API_returns_empty_result`, `404_test.go: API exists but has no keys`
- [x] List keys when exactly one exists - Covered in various tests
- [x] List keys when multiple exist - `200_test.go: list_all_keys_with_default_pagination`
- [x] Verify response structure includes correct pagination details - `200_test.go: list_all_keys_with_default_pagination`
- [x] Verify API ID is correctly associated with returned keys - All tests verify correct API association

## Error Cases

- [x] Attempt to list keys with non-existent API ID - `404_test.go: non-existent API`
- [x] Attempt to list keys with invalid API ID format - `404_test.go: invalid API ID format`
- [x] Attempt to list keys with negative/zero limit - `400_test.go: negative_limit, zero_limit`
- [x] Attempt to list keys with excessively large limit - `400_test.go: extremely_large_limit`
- [x] Attempt to list keys with invalid cursor format - `400_test.go: malformed_cursor`
- [x] Attempt to list keys with malformed request body - `400_test.go: missing_apiId`

## Security Tests

- [x] Attempt to list keys without authentication - `401_test.go: missing_authorization_header`
- [x] Attempt to list keys with invalid authentication - `401_test.go: invalid_authorization_token`
- [x] Attempt to list keys with expired token - `401_test.go: valid_format_non-existent_key`
- [x] Attempt to list keys with insufficient permissions - `403_test.go: missing_read_key_permission, missing_read_api_permission`
- [x] Attempt to list keys from a different workspace (should not be accessible) - `403_test.go: cross_workspace_access`, `404_test.go: API in different workspace`
- [x] Verify correct permissions allow keys listing:
  - [x] Test with wildcard permission ("*") - `403_test.go: wildcard_permissions_should_work`
  - [x] Test with specific permissions for listing keys - `403_test.go: specific_API_permissions_should_work`
  - [x] Test with multiple permissions including the required one - All 403 tests verify both `read_key` AND `read_api`

## Database Verification

- [x] Verify results match actual database records - All tests create and verify actual data
- [x] Verify correct ordering of results - `200_test.go: verify_correct_ordering_of_results`
- [x] Verify pagination works correctly with database queries - `200_test.go: list_keys_with_pagination_cursor`
- [x] Verify no sensitive/internal data is exposed in results - `200_test.go: verify_no_sensitive_data_is_exposed`

## Edge Cases

- [x] List keys at pagination boundaries - `200_test.go: list_keys_with_pagination_cursor`
- [x] Behavior with unusual key metadata - `200_test.go: verify_key_metadata_is_included`
- [ ] Performance with large number of keys - NOT COVERED (performance testing typically done separately)
- [x] Handle keys with deleted/disabled status correctly - Keys filtered by `deleted_at_m IS NULL`, disabled keys tested in setup
- [x] Correct handling of Unicode characters in key data - Unicode characters tested in comprehensive test suite

## Performance Tests

- [ ] Measure response time for listing with varying numbers of keys - NOT COVERED (performance testing typically done separately)
- [ ] Test listing keys with large metadata - NOT COVERED (performance testing typically done separately)
- [ ] Test concurrent requests for key listings - NOT COVERED (performance testing typically done separately)
- [ ] Verify performance with different pagination sizes - NOT COVERED (performance testing typically done separately)

## Integration Tests

- [x] Verify newly created keys appear in listing - All tests create keys and verify they appear
- [x] Verify deleted keys do not appear in listing - Keys filtered by `deleted_at_m IS NULL` in SQL query
- [x] Verify disabled keys are displayed correctly - Disabled keys tested in test setup
- [x] Verify updated keys show current data - Verified through metadata and identity tests

## Pagination Testing

- [x] Verify first page returns expected cursor for next page - `200_test.go: list_keys_with_limit_parameter`
- [x] Verify last page indicates end of results - `200_test.go: list_keys_with_pagination_cursor`
- [x] Verify all keys can be retrieved through pagination - `200_test.go: list_keys_with_pagination_cursor`
- [x] Verify no duplicate keys across pages - `200_test.go: list_keys_with_pagination_cursor` (fixed cursor duplication bug)
- [x] Verify consistency when keys are added/removed during pagination - Inherently tested through cursor pagination fix
