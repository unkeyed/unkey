//go:build !linux

package network

// PinDir returns "" because no eBPF map is pinned off Linux.
func PinDir() string { return "" }

// NewReader returns the no-op reader. Heimdall measures network bytes only on
// Linux; this keeps builds and tests working on other platforms.
func NewReader(string) (Reader, error) { return NewNoop(), nil }
