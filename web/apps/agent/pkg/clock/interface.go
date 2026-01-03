package clock

import "time"

// Clock is an interface for getting the current time.
// We're mainly using this for testing purposes, where waiting in real time
// would be impractical.
type Clock interface {
	Now() time.Time
}
