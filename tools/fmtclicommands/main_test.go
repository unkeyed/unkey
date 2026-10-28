package main

import (
	"bytes"
	"flag"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunCheckWriteAndSkipTests(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "command.go")
	source := []byte(sourceHeader + `var command = &cli.Command{Name: "root"}`)
	require.NoError(t, os.WriteFile(path, source, 0o600))
	require.NoError(t, os.Mkdir(filepath.Join(root, "testdata"), 0o700))
	require.NoError(t, os.Mkdir(filepath.Join(root, "testutil"), 0o700))
	skipped := []string{
		filepath.Join(root, "command_test.go"),
		filepath.Join(root, "testdata", "command.go"),
		filepath.Join(root, "testutil", "command.go"),
	}
	for _, path := range skipped {
		require.NoError(t, os.WriteFile(path, source, 0o600))
	}

	var diagnostics bytes.Buffer
	require.ErrorIs(t, run([]string{root}, &diagnostics), errFormatting)
	require.Contains(t, diagnostics.String(), path)
	require.Contains(t, diagnostics.String(), "mise run fmt-cli-commands")
	unchanged, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, source, unchanged)
	require.ErrorIs(t, run([]string{"--check", root}, io.Discard), errFormatting)
	require.NoError(t, run([]string{"--write", root}, io.Discard))
	require.NoError(t, run([]string{"--check", root}, io.Discard))
	formatted, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(formatted), "&cli.Command{\n\tName: \"root\",\n}")
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	for _, path := range skipped {
		unchanged, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, source, unchanged)
	}
}

func TestRunRejectsConflictingFlagsAndInvalidSource(t *testing.T) {
	require.Error(t, run([]string{"--check", "--write"}, io.Discard))
	require.ErrorIs(t, run([]string{"--help"}, io.Discard), flag.ErrHelp)
	path := filepath.Join(t.TempDir(), "invalid.go")
	source := []byte("package sample\nvar command = &cli.Command{")
	require.NoError(t, os.WriteFile(path, source, 0o600))
	err := run([]string{"--write", path}, io.Discard)
	require.Error(t, err)
	require.Contains(t, err.Error(), path)
	unchanged, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, source, unchanged)
	require.ErrorIs(t, run([]string{filepath.Join(t.TempDir(), "missing.go")}, io.Discard), os.ErrNotExist)
}
