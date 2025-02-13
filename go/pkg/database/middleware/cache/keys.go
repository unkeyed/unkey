package cache

type KeyRatelimitNamespaceByName struct {
	WorkspaceID   string
	NamespaceName string
}

type KeyRatelimitOverride struct {
	WorkspaceID string
	Identifier  string
	NamespaceID string
}
