// Package gdprerasure implements GdprErasureWorkflow, the Restate workflow that
// executes a GDPR erasure request for one subject: a dashboard user, a whole
// workspace, or a customer's end user.
//
// The erasure policy is data, not code paths. registry.go lists, per subject,
// the ordered steps that delete or pseudonymize the subject's rows, and the
// executor runs them generically. MySQL steps delete in batches of at most
// batchRowsMax rows, one restate.Run per batch, until a batch comes back short:
// PlanetScale rejects statements touching more than 100,000 rows, and a short
// batch bounds row locks and replication lag. ClickHouse steps count matching
// rows and then delete them in one restate.Run.
//
// Every step derives its rows from current state, so a crash at any point
// resumes cleanly and a second request for the same subject finds nothing left.
package gdprerasure
