//go:build bpf_generate

package main

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang -target bpfel -tags linux -type observation bpf bpf/hooks.c
