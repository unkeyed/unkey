package notification

// DestinationTestID identifies one test of a saved destination. Reusing a key
// never creates another delivery, even after the destination changes.
func DestinationTestID(workspaceID, destinationID, idempotencyKey string) string {
	return stableID("ntest", workspaceID, destinationID, idempotencyKey)
}
