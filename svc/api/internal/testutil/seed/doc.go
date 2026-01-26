// Package seed provides database seeding utilities for integration tests.
//
// This package handles creating test data in the database with proper relationships
// between entities. It generates unique IDs, handles foreign key constraints, and
// provides sensible defaults while allowing full customization.
//
// # Key Types
//
// [Seeder] is the main type that provides methods to create test entities. It holds
// a database connection and vault service for encrypting keys. [Resources] contains
// the baseline entities created during initial seeding.
//
// # Usage
//
// The seeder is typically used through [testutil.Harness], which wraps it with
// context management. For direct usage:
//
//	seeder := seed.New(t, database, vaultService)
//	seeder.Seed(ctx)  // Creates baseline data
//
//	api := seeder.CreateAPI(ctx, seed.CreateApiRequest{
//	    WorkspaceID: seeder.Resources.UserWorkspace.ID,
//	})
//
//	key := seeder.CreateKey(ctx, seed.CreateKeyRequest{
//	    WorkspaceID: api.WorkspaceID,
//	    KeySpaceID:  api.KeyAuthID.String,
//	    Permissions: []seed.CreatePermissionRequest{{Name: "read", Slug: "read", WorkspaceID: api.WorkspaceID}},
//	})
//
// # Entity Relationships
//
// The seeder handles cascading entity creation. For example, [CreateKeyRequest] can
// include permissions, roles, and rate limits which are created and linked automatically.
// Similarly, [CreateRoleRequest] can include permissions to attach.
//
// # Request Types
//
// Each Create* method has a corresponding request struct that documents all available
// options. Required fields are typically WorkspaceID and identifiers. Optional fields
// use pointers to distinguish between "not set" and "set to zero value".
package seed
