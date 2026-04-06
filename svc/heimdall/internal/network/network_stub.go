//go:build !linux

package network

import "k8s.io/apimachinery/pkg/types"

// stubReader is the macOS / non-Linux implementation: it satisfies the
// Reader interface but does nothing. Heimdall only runs on Linux in
// production; this exists so `go build`, `go test`, and gopls all stay
// green on developer macOS machines.
type stubReader struct{}

func NewReader(_ string) (Reader, error)              { return stubReader{}, nil }
func (stubReader) Attach(_ types.UID) error           { return nil }
func (stubReader) Detach(_ types.UID)                 {}
func (stubReader) Read(_ types.UID) (Counters, error) { return zeroCounters, nil }
func (stubReader) MapEntries() int                    { return 0 }
func (stubReader) Close() error                       { return nil }
