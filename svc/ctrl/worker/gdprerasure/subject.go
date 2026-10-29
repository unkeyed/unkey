package gdprerasure

// subject holds the ids a subject's steps erase by.
type subject struct {
	workspaceID      string
	externalID       string
	userID           string
	email            string
	identityIDs      []string
	orgID            string
	stripeCustomerID string
	depotProjectIDs  []string
}
