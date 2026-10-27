package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

const manifest = "kind: ConfigMap\nmetadata:\n  name: implicit\n---\nkind: Secret\nmetadata:\n  name: explicit\n  namespace: other\n---\nkind: ClusterRole\nmetadata:\n  name: global\n"
const liveObjects = "kind: List\nitems:\n- kind: ConfigMap\n  metadata:\n    name: implicit\n    namespace: release-ns\n    resourceVersion: '42'\n"

type lookup struct {
	ConfigPath string
	Config     json.RawMessage
	Mode       uint32
	Manifest   string
}

func TestMain(m *testing.M) {
	switch name := filepath.Base(os.Args[0]); name {
	case "helm-resource":
		main()
		os.Exit(0)
	case "helm", "kubectl":
		if err := fakeCommand(name, os.Args[1:]); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(17)
		}
		os.Exit(0)
	default:
		os.Exit(m.Run())
	}
}

func TestApplyReturnsLiveObjectsAndPreservesHelmFlags(t *testing.T) {
	dir := setup(t)
	stdout, stderr, err := runCLI(t, dir, "apply", "release", "release-ns", "chart", "--values", "a file.yaml", "--create-namespace")
	require.NoError(t, err, stderr)
	require.Equal(t, liveObjects, stdout)
	require.Contains(t, stderr, "helm progress")
	require.Contains(t, stderr, "kubectl progress")
	require.NotContains(t, stderr, "private-token")

	var helmArgs []string
	readJSON(t, filepath.Join(dir, "helm-args.json"), &helmArgs)
	require.Equal(t, []string{"upgrade", "--install", "release", "chart", "--namespace", "release-ns", "--values", "a file.yaml", "--create-namespace", "--output", "json"}, helmArgs)

	var got lookup
	readJSON(t, filepath.Join(dir, "lookup.json"), &got)
	require.JSONEq(t, configJSON(t, "release-ns"), string(got.Config))
	require.Equal(t, uint32(0600), got.Mode)
	require.Equal(t, manifest, got.Manifest)
	require.NoFileExists(t, got.ConfigPath)
	require.FileExists(t, filepath.Join(dir, "release"))
}

func TestCommandFailuresNeverEmitPartialObjectsOrLeaveCredentials(t *testing.T) {
	for _, stage := range []string{"helm", "config", "kubectl"} {
		t.Run(stage, func(t *testing.T) {
			dir := setup(t)
			t.Setenv("FAIL_STAGE", stage)
			stdout, stderr, err := runCLI(t, dir, "apply", "release", "release-ns", "chart")
			require.Error(t, err)
			require.Empty(t, stdout)
			require.Contains(t, stderr, "exit status 17")
			require.NotContains(t, stderr, "private-token")
			files, err := filepath.Glob(filepath.Join(dir, "helm-resource-kubeconfig-*"))
			require.NoError(t, err)
			require.Empty(t, files)
		})
	}
}

func TestRejectsInvalidArguments(t *testing.T) {
	for _, args := range [][]string{nil, {"unknown", "r", "ns"}, {"apply", "r", "ns"}, {"delete", "", "ns"}, {"apply", "r", "ns", "--version"}} {
		t.Run(fmt.Sprint(args), func(t *testing.T) {
			dir := setup(t)
			stdout, stderr, err := runCLI(t, dir, args...)
			require.Error(t, err)
			require.Empty(t, stdout)
			require.NotEmpty(t, stderr)
			require.NoFileExists(t, filepath.Join(dir, "helm-args.json"))
		})
	}
}

func TestRejectsMalformedReleaseAndConfigWithoutLeakingCredentials(t *testing.T) {
	for _, test := range []struct {
		name, variable, value string
	}{
		{"invalid release JSON", "RELEASE_JSON", "not JSON"},
		{"missing manifest", "RELEASE_JSON", `{}`},
		{"wrong manifest type", "RELEASE_JSON", `{"manifest":17}`},
		{"empty manifest", "RELEASE_JSON", `{"manifest":""}`},
		{"invalid config JSON", "CONFIG_JSON", "private-token bad JSON"},
		{"missing contexts", "CONFIG_JSON", `{}`},
		{"multiple contexts", "CONFIG_JSON", `{"contexts":[{},{}]}`},
		{"null context", "CONFIG_JSON", `{"contexts":[{"name":"test","context":null}],"current-context":"test"}`},
		{"wrong current context", "CONFIG_JSON", `{"contexts":[{"name":"test","context":{}}],"current-context":"other"}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := setup(t)
			t.Setenv(test.variable, test.value)
			stdout, stderr, err := runCLI(t, dir, "apply", "release", "release-ns", "chart")
			require.Error(t, err)
			require.Empty(t, stdout)
			require.NotEmpty(t, stderr)
			require.NotContains(t, stderr, "private-token")
		})
	}
}

func TestDeleteIsIdempotentAndKeepsDiagnosticsOffStdout(t *testing.T) {
	dir := setup(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "release"), nil, 0600))
	for range 2 {
		stdout, stderr, err := runCLI(t, dir, "delete", "release", "release-ns", "--wait")
		require.NoError(t, err, stderr)
		require.Empty(t, stdout)
		require.Contains(t, stderr, "release uninstalled")
		require.NoFileExists(t, filepath.Join(dir, "release"))
	}
	var helmArgs []string
	readJSON(t, filepath.Join(dir, "helm-args.json"), &helmArgs)
	require.Equal(t, []string{"uninstall", "release", "--namespace", "release-ns", "--ignore-not-found", "--wait"}, helmArgs)
	t.Setenv("FAIL_STAGE", "helm")
	stdout, stderr, err := runCLI(t, dir, "delete", "release", "release-ns")
	require.Error(t, err)
	require.Empty(t, stdout)
	require.Contains(t, stderr, "helm uninstall")
}

func setup(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	executable, err := os.Executable()
	require.NoError(t, err)
	for _, name := range []string{"helm-resource", "helm", "kubectl"} {
		require.NoError(t, os.Symlink(executable, filepath.Join(dir, name)))
	}
	t.Setenv("PATH", dir)
	t.Setenv("TMPDIR", dir)
	t.Setenv("TEST_STATE", dir)
	t.Setenv("CONFIG_JSON", configJSON(t, "original"))
	t.Setenv("FAIL_STAGE", "")
	t.Setenv("RELEASE_JSON", "")
	return dir
}

func runCLI(t *testing.T, dir string, args ...string) (string, string, error) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), filepath.Join(dir, "helm-resource"), args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func readJSON(t *testing.T, path string, target any) {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, target))
}

func configJSON(t *testing.T, namespace string) string {
	t.Helper()
	data, err := json.Marshal(map[string]any{
		"current-context": "test",
		"contexts": []any{map[string]any{
			"name":    "test",
			"context": map[string]string{"cluster": "cluster", "user": "user", "namespace": namespace},
		}},
		"users": []any{map[string]any{
			"name": "user",
			"user": map[string]string{"token": "private-token"},
		}},
		"clusters": []any{map[string]any{
			"name":    "cluster",
			"cluster": map[string]string{"server": "https://cluster"},
		}},
		"extensions": []any{map[string]any{"name": "preserved", "extension": map[string]bool{"enabled": true}}},
	})
	require.NoError(t, err)
	return string(data)
}

func fakeCommand(name string, args []string) error {
	if _, err := fmt.Fprintln(os.Stderr, name+" progress"); err != nil {
		return err
	}
	stage := name
	if name == "kubectl" && len(args) > 0 && args[0] == "config" {
		stage = "config"
	}
	if os.Getenv("FAIL_STAGE") == stage {
		if _, err := fmt.Print("partial output must not reach Tilt"); err != nil {
			return err
		}
		return errors.New("command failed")
	}
	if name == "helm" {
		if err := writeJSON("helm-args.json", args); err != nil {
			return err
		}
		if args[0] == "uninstall" {
			err := os.Remove(filepath.Join(os.Getenv("TEST_STATE"), "release"))
			if err != nil && !(errors.Is(err, os.ErrNotExist) && slices.Contains(args, "--ignore-not-found")) {
				return err
			}
			_, err = fmt.Println("release uninstalled")
			return err
		}
		if args[0] != "upgrade" || !slices.Contains(args, "--install") {
			return errors.New("expected a Helm release installation")
		}
		if err := os.WriteFile(filepath.Join(os.Getenv("TEST_STATE"), "release"), nil, 0600); err != nil {
			return err
		}
		if value := os.Getenv("RELEASE_JSON"); value != "" {
			_, err := fmt.Print(value)
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(struct {
			Manifest string `json:"manifest"`
		}{Manifest: manifest})
	}
	if stage == "config" {
		_, err := fmt.Print(os.Getenv("CONFIG_JSON"))
		return err
	}
	if len(args) != 6 || args[0] != "--kubeconfig" || !slices.Equal(args[2:], []string{"get", "-oyaml", "-f", "-"}) {
		return fmt.Errorf("unexpected kubectl lookup: %q", args)
	}
	config, err := os.ReadFile(args[1])
	if err != nil {
		return err
	}
	info, err := os.Stat(args[1])
	if err != nil {
		return err
	}
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return err
	}
	if err := writeJSON("lookup.json", lookup{ConfigPath: args[1], Config: config, Mode: uint32(info.Mode().Perm()), Manifest: string(input)}); err != nil {
		return err
	}
	_, err = fmt.Print(liveObjects)
	return err
}

func writeJSON(name string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(os.Getenv("TEST_STATE"), name), data, 0600)
}
