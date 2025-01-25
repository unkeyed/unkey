package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/config"
)

func TestLoadFile_WithMissingRequired(t *testing.T) {

	cfg := struct {
		Hello string `json:"hello" required:"true"`
	}{Hello: ""}

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	err := os.WriteFile(path, []byte(`{"somethingElse": "world"}`), 0644)
	require.NoError(t, err)

	err = config.LoadFile(&cfg, path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "hello is required")

}

func TestLoadFile_WritesValuesToPointer(t *testing.T) {

	cfg := struct {
		Hello string `json:"hello" required:"true"`
	}{Hello: ""}

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	err := os.WriteFile(path, []byte(`{"hello": "world"}`), 0644)
	require.NoError(t, err)

	err = config.LoadFile(&cfg, path)
	require.NoError(t, err)
	require.Equal(t, "world", cfg.Hello)

}

func TestLoadFile_ExpandsEnv(t *testing.T) {

	cfg := struct {
		Hello string `json:"hello" required:"true"`
	}{
		Hello: "",
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	err := os.WriteFile(path, []byte(`{"hello": "${TEST_HELLO}"}`), 0644)
	require.NoError(t, err)

	t.Setenv("TEST_HELLO", "world")
	err = config.LoadFile(&cfg, path)
	require.NoError(t, err)
	require.Equal(t, "world", cfg.Hello)

}
