package cache

type KeyRatelimitNamespaceByName struct {
	WorkspaceID   string
	NamespaceName string
}

type KeyRatelimitOverridesByIdentifier struct {
	WorkspaceID string
	Identifier  string
	NamespaceID string
}

type KeyRatelimitOverrideByID struct {
	WorkspaceID string
	OverrideID  string
}
