# Test Scenarios for v2_permissions_create_role

This document outlines test scenarios for the API endpoint that creates a new role in the system.

## Happy Path Scenarios

- [x] Create a basic role with a valid name
- [x] Create role with a description
- [x] Create role without description
- [x] Create multiple roles with different names
- [x] Verify roleId is returned in the response
- [ ] Create role with maximum length name
- [ ] Create role with special characters in the name (if supported)

Note: Permission assignment is handled by separate endpoints, not during role creation.

## Error Cases

- [x] Attempt to create role with empty name
- [x] Attempt to create role with duplicate name (should return appropriate error)
- [x] Attempt to create role with invalid JSON in request
- [x] Attempt to create role with missing required fields
- [x] Attempt to create role with malformed request body
- [x] Attempt to create role with very long description
- [ ] Attempt to create role with name shorter than minimum length

## Security Tests

- [x] Attempt to create role without authentication
- [x] Attempt to create role with invalid authentication
- [x] Attempt to create role with malformed authorization header
- [x] Attempt to create role with insufficient permissions
- [x] Verify correct permissions allow role creation:
  - [x] Test with specific permission for creating roles (rbac.*.create_role)
- [x] Test workspace isolation (roles created in correct workspace)
- [ ] Attempt to create role with expired token

## Database Verification

- [x] Verify role record is correctly inserted in the database
- [x] Verify role has the correct workspace ID
- [x] Verify role has correct name and description
- [x] Verify role has non-null created timestamp
- [x] Verify audit log entry is created for role creation

Note: Role-permission relationships are managed separately via other endpoints.

## Edge Cases

- [x] Create role with description at maximum length (validation test)
- [x] Create role with similar name to existing role (case sensitivity test)
- [ ] Create role with name at exactly minimum length
- [ ] Create role with name at exactly maximum length
- [ ] Create role with Unicode characters in name/description

## Performance Tests

- [ ] Measure response time for role creation
- [ ] Test creating multiple roles in parallel
- [ ] Test system behavior under load
- [ ] Test creating role with large number of permissions

## Integration Tests

- [ ] Verify ability to use the created role in key assignment
- [ ] Verify ability to assign the role to users/identities
- [ ] Verify role appears in role listing endpoints
- [ ] Verify role can be retrieved by its ID
- [ ] Verify permissions can be assigned to the role via separate endpoints
- [ ] Verify permissions are correctly evaluated when using the role

Note: These integration tests require other endpoints to be implemented (role assignment, permission assignment, etc.)