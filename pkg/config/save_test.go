package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type testConfig struct {
	Name  string `toml:"name"`
	Count int    `toml:"count"`
}

func TestSave_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	original := testConfig{Name: "test", Count: 42}
	err := Save(path, original)
	require.NoError(t, err)

	loaded, err := Load[testConfig](path)
	require.NoError(t, err)
	require.Equal(t, original, loaded)
}

func TestSave_CreatesDirectories(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "deep", "config.toml")

	err := Save(path, testConfig{Name: "test"})
	require.NoError(t, err)

	_, statErr := os.Stat(path)
	require.NoError(t, statErr)
}

func TestSave_FilePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	err := Save(path, testConfig{})
	require.NoError(t, err)

	info, statErr := os.Stat(path)
	require.NoError(t, statErr)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestSave_RejectsNonToml(t *testing.T) {
	err := Save("/tmp/config.json", testConfig{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "only .toml files are supported")
}
