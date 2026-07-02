// Package deployteardown implements DeployTeardownService, the Restate virtual
// object that stops all of a workspace's running Deploy compute and confirms it
// drained.
//
// It exists because canceling a Deploy subscription removed billing before
// stopping compute: usage kept accruing with nowhere to land. Teardown fixes the
// ordering. Stop compute first and billing takes care of itself: once compute
// drains usage freezes, and the existing hourly deploybilling push bills the
// final total at the normal month-end invoice.
//
// One primitive serves two callers. Cancel tears down with ARCHIVE (permanent);
// the spend cap (ENG-2923) with SUSPEND (resumable). They differ only in the
// desired state the stopped deployments land in.
//
// The object is keyed by workspace id, so each workspace's teardowns serialize
// and a stuck drain in one workspace cannot block another's.
package deployteardown
