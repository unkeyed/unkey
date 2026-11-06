// Package privatenetwork maintains Ctrl's per-platform connection catalogs.
//
// Consumers must honor [Snapshot.Certified] before counting missing grants
// toward deletion. Certification needs recent single-shard database progress
// and synchronized source and Ctrl clocks. Stream heartbeats prove liveness,
// not progress. Multi-shard streams update catalogs but never certify deletions.
package privatenetwork
