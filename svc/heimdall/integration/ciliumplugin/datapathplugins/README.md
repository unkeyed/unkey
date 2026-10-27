# Cilium datapath-plugin protocol

These protocol and generated Go files are copied without changes from Cilium
v1.20.2, commit `e0dc92bd6dac33d7ba3b5eace82a0448bbb13aac`, directory
`api/v1/datapathplugins`. The Apache-2.0 license is included.

Only the protobuf messages and gRPC bindings are used. This avoids importing
Cilium's agent and its dependency graph. The optional JSON bindings are omitted.

The diagnostic server accepts only Cilium 1.20.2. To change that version, review
the upstream protocol and loader, replace these files from the same revision,
and rerun the local integration. Cilium does not guarantee plugin compatibility
between releases.
