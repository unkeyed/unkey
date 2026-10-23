package domainverify

import "time"

const (
	// RetryInterval is how often Restate re-runs a verification invocation.
	RetryInterval = time.Minute

	// DurationMax is how long a domain may stay unverified before its row is
	// marked failed.
	DurationMax = 24 * time.Hour

	// RowCommitGrace is how long a missing row stays retryable. The invocation
	// is sent inside the transaction that inserts the row, so it can run before
	// the commit lands.
	RowCommitGrace = 2 * time.Minute
)

// RetryAttemptsMax outlasts [DurationMax] by an hour so the handler marks the
// row failed before Restate pauses the invocation, which would hold the key.
const RetryAttemptsMax = uint((DurationMax + time.Hour) / RetryInterval)
