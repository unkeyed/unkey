package codes

// Resource-specific error categories

// dataKey defines errors related to API key operations.
type dataKey struct {
	// NotFound indicates the requested key was not found.
	NotFound Code
}

// dataWorkspace defines errors related to workspace operations.
type dataWorkspace struct {
	// NotFound indicates the requested workspace was not found.
	NotFound Code
}

// dataApi defines errors related to API operations.
type dataApi struct {
	// NotFound indicates the requested API was not found.
	NotFound Code
}

// dataMigration defines errors related to migration operations.
type dataMigration struct {
	// NotFound indicates the requested migration was not found.
	NotFound Code
}

// dataPermission defines errors related to permission operations.
type dataPermission struct {
	// Duplicate indicates the requested permission already exists.
	Duplicate Code

	// NotFound indicates the requested permission was not found.
	NotFound Code
}

// dataRole defines errors related to role operations.
type dataRole struct {
	// Duplicate indicates the requested role already exists.
	Duplicate Code

	// NotFound indicates the requested role was not found.
	NotFound Code
}

// dataKeyAuth defines errors related to key authentication operations.
type dataKeyAuth struct {
	// NotFound indicates the requested key authentication was not found.
	NotFound Code
}

// dataRatelimitNamespace defines errors related to rate limit namespace operations.
type dataRatelimitNamespace struct {
	// NotFound indicates the requested rate limit namespace was not found.
	NotFound Code
}

// dataRatelimitOverride defines errors related to rate limit override operations.
type dataRatelimitOverride struct {
	// NotFound indicates the requested rate limit override was not found.
	NotFound Code
}

// dataIdentity defines errors related to identity operations.
type dataIdentity struct {
	// NotFound indicates the requested identity was not found.
	NotFound Code

	// Duplicate indicates the requested identity already exists.
	Duplicate Code
}

// dataAuditLog defines errors related to audit log operations.
type dataAuditLog struct {
	// NotFound indicates the requested audit log was not found.
	NotFound Code
}

// UnkeyDataErrors defines all data-related errors in the Unkey system.
// These errors generally relate to CRUD operations on domain entities.
type UnkeyDataErrors struct {
	// Resource-specific categories
	Key                dataKey
	Workspace          dataWorkspace
	Api                dataApi
	Migration          dataMigration
	Permission         dataPermission
	Role               dataRole
	KeyAuth            dataKeyAuth
	RatelimitNamespace dataRatelimitNamespace
	RatelimitOverride  dataRatelimitOverride
	Identity           dataIdentity
	AuditLog           dataAuditLog
}

// Data contains all predefined data-related error codes.
// These errors can be referenced directly (e.g., codes.Data.Key.NotFound)
// for consistent error handling throughout the application.
var Data = UnkeyDataErrors{
	Key: dataKey{
		NotFound: Code{SystemUnkey, CategoryUnkeyData, "key_not_found"},
	},

	Workspace: dataWorkspace{
		NotFound: Code{SystemUnkey, CategoryUnkeyData, "workspace_not_found"},
	},

	Api: dataApi{
		NotFound: Code{SystemUnkey, CategoryUnkeyData, "api_not_found"},
	},

	Migration: dataMigration{
		NotFound: Code{SystemUnkey, CategoryUnkeyData, "migration_not_found"},
	},

	Permission: dataPermission{
		NotFound:  Code{SystemUnkey, CategoryUnkeyData, "permission_not_found"},
		Duplicate: Code{SystemUnkey, CategoryUnkeyData, "permission_already_exists"},
	},

	Role: dataRole{
		NotFound:  Code{SystemUnkey, CategoryUnkeyData, "role_not_found"},
		Duplicate: Code{SystemUnkey, CategoryUnkeyData, "role_already_exists"},
	},

	KeyAuth: dataKeyAuth{
		NotFound: Code{SystemUnkey, CategoryUnkeyData, "key_auth_not_found"},
	},

	RatelimitNamespace: dataRatelimitNamespace{
		NotFound: Code{SystemUnkey, CategoryUnkeyData, "ratelimit_namespace_not_found"},
	},

	RatelimitOverride: dataRatelimitOverride{
		NotFound: Code{SystemUnkey, CategoryUnkeyData, "ratelimit_override_not_found"},
	},

	Identity: dataIdentity{
		NotFound:  Code{SystemUnkey, CategoryUnkeyData, "identity_not_found"},
		Duplicate: Code{SystemUnkey, CategoryUnkeyData, "identity_already_exists"},
	},

	AuditLog: dataAuditLog{
		NotFound: Code{SystemUnkey, CategoryUnkeyData, "audit_log_not_found"},
	},
}
