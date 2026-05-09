package codes

// portalSession defines errors related to portal session operations.
type portalSession struct {
	// TokenMissing indicates a portal session token was not provided.
	TokenMissing Code

	// SessionNotFound indicates the portal session was not found or has expired.
	SessionNotFound Code
}

// UnkeyPortalErrors defines all portal-related error codes.
type UnkeyPortalErrors struct {
	// Session contains errors related to portal session operations.
	Session portalSession
}

// Portal contains all predefined portal error codes.
var Portal = UnkeyPortalErrors{
	Session: portalSession{
		TokenMissing:    Code{SystemUnkey, CategoryUnkeyAuthentication, "portal_token_missing", KindUnknown},
		SessionNotFound: Code{SystemUnkey, CategoryUnkeyAuthentication, "portal_session_not_found", KindUnknown},
	},
}
