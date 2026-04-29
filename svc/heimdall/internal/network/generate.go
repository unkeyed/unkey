//go:build bpf_generate

// This file hosts the `go:generate` directive for bpf2go. It's gated
// behind the `bpf_generate` build tag so that `make generate` (which runs
// `go generate ./...` across the whole repo) does NOT invoke bpf2go and
// therefore does not require clang / llvm-strip on the machine. Run
// `make generate-bpf` to regenerate the BPF bindings after editing the
// C source.
//
// Regenerating requires clang 12+ (brew install llvm). bpf/network.bpf.c
// is self-contained against bpf/network_helpers.h, so no kernel headers,
// no vmlinux.h, no libbpf install. If a new BPF helper or kernel struct
// is needed, add it to network_helpers.h.
//
// The bpfel target (little-endian) covers x86_64 and arm64, the only
// architectures we deploy to. If we ever ship to a big-endian platform,
// add a parallel `bpfeb` directive.
//
// See doc.go for the package overview.

package network

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang -target bpfel -type counters bpf bpf/network.bpf.c -- -I./bpf
