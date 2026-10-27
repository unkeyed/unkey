// Package deploymentnotify freezes deployment lifecycle reports into notification
// events before dispatch. Its deployment-keyed object suppresses stale changes and
// advances lifecycle state only after dispatch succeeds. Snapshot data and time
// are journaled together so replay keeps the same event contents.
package deploymentnotify
