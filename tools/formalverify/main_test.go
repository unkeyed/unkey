package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommandExploresDeploymentContracts(t *testing.T) {
	t.Chdir("../..")
	var out bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"--check", "deploygate", "--json"}, &out))
	var results []result
	require.NoError(t, json.Unmarshal(out.Bytes(), &results))
	require.Len(t, results, 1)
	require.Equal(t, "deploygate", results[0].Name)
	require.Equal(t, 2432, results[0].Cases)
	require.Zero(t, results[0].Violations)
	require.Empty(t, results[0].Counterexample)
}

func TestCommandRejectsUnknownChecks(t *testing.T) {
	var out bytes.Buffer
	require.Error(t, run(t.Context(), []string{"--check", "typo"}, &out))
	require.Empty(t, out.String())
}

func TestCommandRejectsSourceDrift(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "tools/formalverify"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "source.go"), []byte("changed source"), 0o600))
	manifest := fmt.Sprintf("%x  source.go\n", sha256.Sum256([]byte("reviewed source")))
	require.NoError(t, os.WriteFile(filepath.Join(root, "tools/formalverify/sources.sha256"), []byte(manifest), 0o600))
	t.Chdir(root)
	var out bytes.Buffer
	require.ErrorContains(t, run(t.Context(), []string{"--check", "deploygate"}, &out), "source drift: source.go")
	require.Empty(t, out.String())
}

func TestCommandCatchesRealSourceMutations(t *testing.T) {
	t.Chdir("../..")
	var out bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"--self-test"}, &out))
	require.Contains(t, out.String(), "deploygate mutation caught:")
	require.Contains(t, out.String(), "circuitbreaker-recovery mutation caught:")
}
