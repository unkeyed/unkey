# Local Cilium plugin integration

This diagnostic executable instruments `cil_from_container` in Cilium 1.20.2.
It runs separately from Heimdall's collector. It does not change billing,
ClickHouse, pod-side hooks, or production deployment configuration.

Both hooks return `TC_ACT_UNSPEC`. PRE counts invocations. POST counts the
original Cilium return value: pass, drop, redirect, or other. The hooks do not
pair packets through shared scratch state, read packet bytes, or modify packets.
A redirect verdict is not proof of delivery. This measures this entrypoint,
not all Cilium drops or all later local drops.

## Run the isolated integration

Run from the repository root on a Linux amd64 host with Docker and the pinned mise
tools. `start.sh` uses only the `heimdall-cilium` Minikube profile and a dedicated
kubeconfig. It leaves the cluster running for inspection. Do not use this
registration on a shared cluster: the plugin resource is cluster-scoped, and
version changes reload the cluster datapath.

```bash
mise exec -- bash svc/heimdall/integration/ciliumplugin/start.sh
mise exec -- bash svc/heimdall/integration/ciliumplugin/check.sh
```

The fixture permits client egress to one HTTP server and denies it to another.
An unfiltered client checks that the denied server is otherwise reachable.
The check requires both hooks to execute, redirects for successful traffic,
and drops for denied traffic. It checks the actual TCX program, freplace target
IDs, and observation map references rather than accepting an RPC success as
attachment evidence.

The local topology uses native routing, no masquerading, no kube-proxy
replacement, and a single node. It is not an AWS CNI chaining, gVisor, IPv6,
cross-node, or production-throughput test.

Regenerate the embedded BPF object after changing C source:

```bash
mise run generate-bpf
```

## Interpret observations

Observation maps belong to one Cilium attachment generation. Cilium owns their
pins and the freplace links. Restarting the diagnostic process preserves live
hooks and their maps. Successful endpoint regeneration replaces them and resets
the diagnostic counters. These are not durable billing counters.

`-inspect /sys/fs/bpf/cilium/endpoints/<endpoint-id>` checks a live attachment
and reads its map. A missing map or hook returns an error, not zero. The query
is a point-in-time check, not evidence of uninterrupted coverage. Concurrent
traffic can change counters while they are read.

The RPC supplies namespace, pod name, and interface name, not pod UID or sandbox
identity. Logs and inspection explicitly mark identity as `unverified`. The
namespace flag limits instrumentation, not billing attribution. Use Kubernetes
and containerd independently to establish fixture identity. Production use
requires UID/sandbox validation and coverage reconciliation.

## Exercise lifecycle failures

Use only the dedicated kubeconfig:

```bash
export KUBECONFIG="$PWD/.cache/cilium-integration/kubeconfig"
```

Stop `heimdall-plugin` inside the node, then rerun the traffic check. Existing
hooks must keep working. With the server stopped, change the plugin resource's
`spec.version`. `BestEffort` skips the unavailable Prepare RPC, so traffic can
work with no hooks. Restarting the server alone does not repair that omission.
Change `spec.version` again and verify attachment before reading counters.

`BestEffort` does not ignore Instrument RPC failures or invalid successful
Prepare responses. Those can prevent endpoint regeneration. The server bounds
admitted RPCs and sets a context deadline, but a synchronous BPF syscall is not
interruptible through that context. Do not treat this as an availability bound.

The systemd runtime directory removes stale sockets after process exit. Cilium,
not this executable, cleans staging pins on an instrumentation failure.

To remove the experiment, first delete its registration and verify that the
hooks disappear while traffic still works. Then stop the diagnostic process:

```bash
mise exec -- kubectl delete ciliumdatapathplugin heimdall
mise exec -- docker exec heimdall-cilium systemctl stop heimdall-plugin
```

Stopping the server alone does not remove live hooks. This prototype is not
wired into Heimdall startup and must not be used for invoice credits.
