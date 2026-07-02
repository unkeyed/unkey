package deployteardown

// suspensionKey is the persistent virtual-object state key under which a
// SUSPEND teardown records what it stopped so Resume can restore it.
const suspensionKey = "suspension"

// suspension records what a SUSPEND teardown stopped so Resume can restore it.
type suspension struct {
	// AppCurrent maps app_id -> the deployment_id that was that app's current
	// deployment when it was suspended, so resume re-promotes exactly that one.
	AppCurrent map[string]string
}
