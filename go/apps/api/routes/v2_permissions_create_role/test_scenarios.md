# Test Scenarios for v2_permissions_create_role

This document outlines test scenarios for the API endpoint that creates a new role in the system.

## Happy Path Scenarios

- [ ] Create a basic role with a valid name
- [ ] Create role with a description
- [ ] Create role with permissions
- [ ] Create role with multiple permissions
- [ ] Create multiple roles with different names
- [ ] Create role with maximum length name
- [ ] Verify roleId is returned in the response
- [ ] Create role with special characters in the name (if supported)
- [ ] Create role with wildcard permissions

## Error Cases

- [ ] Attempt to create role with empty name
- [ ] Attempt to create role with name shorter than minimum length
- [ ] Attempt to create role with duplicate name (should return appropriate error)
- [ ] Attempt to create role with non-existent permissions
- [ ] Attempt to create role with invalid JSON in request
- [ ] Attempt to create role with missing required fields
- [ ] Attempt to create role with malformed request body
- [ ] Attempt to create role with too many permissions (if there's a limit)

## Security Tests

- [ ] Attempt to create role without authentication
- [ ] Attempt to create role with invalid authentication
- [ ] Attempt to create role with expired token
- [ ] Attempt to create role with insufficient permissions
- [ ] Verify correct permissions allow role creation:
  - [ ] Test with wildcard permission ("*")
  - [ ] Test with specific permission for creating roles
  - [ ] Test with multiple permissions including the required one
- [ ] Attempt to create role with permissions from another workspace

## Database Verification

- [ ] Verify role record is correctly inserted in the database
- [ ] Verify role has the correct workspace ID
- [ ] Verify role has correct name and description
- [ ] Verify role-permission relationships are correctly stored
- [ ] Verify role has non-null created timestamp
- [ ] Verify audit log entry is created for role creation

## Edge Cases

- [ ] Create role with name at exactly minimum length
- [ ] Create role with name at exactly maximum length
- [ ] Create role with description at maximum length
- [ ] Create role with Unicode characters in name/description
- [ ] Create role with similar name to existing role
- [ ] Create role with maximum number of allowed permissions

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
- [ ] Verify permissions are correctly evaluated when using the role
- [ ] Verify adding new permissions to a role affects authorization checks