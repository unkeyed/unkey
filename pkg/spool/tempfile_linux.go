//go:build linux

package spool

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/unkeyed/unkey/pkg/fault"
	"golang.org/x/sys/unix"
)

func checkDir(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		return fault.Wrap(err, fault.Internal("stat Dir failed"))
	}
	if !info.IsDir() {
		return fault.New(fmt.Sprintf("%s is not a directory", dir))
	}
	var stat unix.Statfs_t
	if err := unix.Statfs(dir, &stat); err != nil {
		return fault.Wrap(err, fault.Internal("statfs Dir failed"))
	}
	if err := checkFilesystem(int64(stat.Type)); err != nil {
		return err
	}
	file, err := openTempFile(dir)
	if err != nil {
		return err
	}
	return closeFile(file)
}

func openTempFile(dir string) (*os.File, error) {
	file, err := openUnnamed(dir)
	if errors.Is(err, unix.EOPNOTSUPP) || errors.Is(err, unix.EISDIR) {
		file, err = createUnlinked(dir)
	}
	if err != nil {
		return nil, err
	}
	var stat unix.Statfs_t
	if err := unix.Fstatfs(int(file.Fd()), &stat); err != nil {
		return nil, errors.Join(fault.Wrap(err, fault.Internal("fstatfs temporary file failed")), closeFile(file))
	}
	if err := checkFilesystem(int64(stat.Type)); err != nil {
		return nil, errors.Join(err, closeFile(file))
	}
	return file, nil
}

// openUnnamed uses O_TMPFILE, so the file never has a name and the kernel
// frees it when the last descriptor closes.
func openUnnamed(dir string) (*os.File, error) {
	fd, err := unix.Open(dir, unix.O_TMPFILE|unix.O_RDWR|unix.O_CLOEXEC, 0o600)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("opening O_TMPFILE failed"))
	}
	return os.NewFile(uintptr(fd), filepath.Join(dir, "(spool)")), nil
}

// createUnlinked is the fallback for filesystems without O_TMPFILE. The file
// is private (0600, exclusive create) and is unlinked before any byte is
// written.
func createUnlinked(dir string) (*os.File, error) {
	file, err := os.CreateTemp(dir, ".spool-*")
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("creating temporary file failed"))
	}
	if err := os.Remove(file.Name()); err != nil {
		return nil, errors.Join(fault.Wrap(err, fault.Internal("unlinking temporary file failed")), closeFile(file))
	}
	return file, nil
}

func checkFilesystem(magic int64) error {
	switch magic {
	case unix.TMPFS_MAGIC:
		return fault.New("Dir is on tmpfs, which is memory-backed")
	case unix.RAMFS_MAGIC:
		return fault.New("Dir is on ramfs, which is memory-backed")
	}
	return nil
}
