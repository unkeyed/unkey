package portal

// Grant is the JSON object stored on a portal session's `scopes` column: the
// simplified capability model that portal.createSession writes and GetSession
// reads back. The portal_session resolver expands the verbs into RBAC
// permission strings via portalrbac.
//
// It lives here, shared by both sides, because it is a wire format rather than
// either side's private struct. MySQL stores it as opaque JSON, so a tag that
// drifted between writer and reader would not fail to compile or to unmarshal;
// it would silently yield a session with no keyspaces and no scopes.
type Grant struct {
	KeyspaceIDs []string `json:"keyspaceIds"`
	Scopes      []string `json:"scopes"`
}
