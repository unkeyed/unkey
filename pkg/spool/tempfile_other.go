//go:build !linux

package spool

import (
	"os"

	"github.com/unkeyed/unkey/pkg/fault"
)

func checkDir(string) error {
	return errUnsupported()
}

func openTempFile(string) (*os.File, error) {
	return nil, errUnsupported()
}

func errUnsupported() error {
	return fault.New("spool requires Linux: the disk-backed storage checks and unnamed temporary files are implemented only there")
}
