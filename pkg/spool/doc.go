// Package spool validates a streamed request body while it copies the body to
// an unnamed temporary file, and returns a replayable [Body] only when the
// validator accepted every byte that was stored.
//
// The package exists so that a proxy can validate large bodies, including
// very large individual JSON strings, without buffering them on the Go heap,
// and so that an invalid body is never forwarded. [Pool.Admit] is the only
// entry point: a [Body] cannot exist unless validation succeeded, the input
// reached EOF, and the stored size matches a declared Content-Length.
//
// # Resource bounds
//
// Each admission uses one fixed 32 KiB copy buffer plus whatever the validator
// buffers. A [Pool] bounds open bodies with [Config.ConcurrentMax] and the
// reserved bytes of all bodies with [Config.TotalBytesMax]. A slot and its
// bytes stay reserved from admission until [Body.Close]. Capacity checks fail
// at once with [FailureCapacity]; the pool never queues.
//
// These are bounds on application buffering, not on the memory of the
// process's cgroup: file writes go through the kernel page cache, which counts
// toward container memory until written back and reclaimed.
//
// # Storage
//
// Bodies are stored in [Config.Dir] as files that have no directory entry
// (O_TMPFILE, or create then unlink at once), so they disappear when the last
// descriptor closes, including after a crash. The package is Linux-only:
// [NewPool] fails on other systems. It rejects tmpfs and ramfs directories and
// rechecks the filesystem of each file. That check does not prove that other
// filesystems, such as FUSE or network mounts, are disk-backed; the operator
// must choose Dir.
package spool
