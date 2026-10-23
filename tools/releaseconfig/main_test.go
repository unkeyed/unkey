package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestReleaseConfigScopesHistoryToReleaseDependencies(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("GOWORK", "off")
	writeFile(t, "go.mod", "module example.test/release\n\ngo 1.26\n")
	writeFile(t, "service.yaml", `version: 2
project_name: app
includes:
  - from_file:
      path: base.yaml
`)
	writeFile(t, "base.yaml", `version: 2
builds:
  - main: ./build/{{ .ProjectName }}
    env: [CGO_ENABLED=0]
    goos: [linux, darwin]
    goarch: [amd64, arm64]
    tags: [cli_release]
dockers_v2:
  - dockerfile: Dockerfile.release
`)
	writeFile(t, "build/app/main.go", "package main\nimport _ \"example.test/release/pkg/shared\"\nfunc main() {}\n")
	writeFile(t, "build/app/release.go", "//go:build cli_release\n\npackage main\nimport _ \"example.test/release/pkg/cli\"\n")
	writeFile(t, "build/app/dev.go", "//go:build !cli_release\n\npackage main\nimport _ \"example.test/release/pkg/dev\"\n")
	writeFile(t, "pkg/shared/shared.go", "package shared\nimport \"embed\"\n//go:embed assets\nvar assets embed.FS\n")
	writeFile(t, "pkg/shared/arch_arm64.go", "package shared\nimport _ \"example.test/release/pkg/arm\"\n")
	writeFile(t, "pkg/shared/os_darwin.go", "package shared\nimport _ \"example.test/release/pkg/mac\"\n")
	for _, name := range []string{"cli", "dev", "arm", "mac"} {
		writeFile(t, "pkg/"+name+"/code.go", "package "+name+"\n")
	}
	writeFile(t, "pkg/shared/assets/data.txt", "before\n")
	writeFile(t, "pkg/shared/deleted.go", "package shared\n")
	git(t, "init", "--quiet")
	git(t, "config", "user.name", "Release Test")
	git(t, "config", "user.email", "release@example.test")
	git(t, "add", ".")
	git(t, "commit", "--quiet", "-m", "base")
	git(t, "tag", "app/v1.0.0")

	changes := []struct {
		path    string
		content string
		include bool
	}{
		{"web/page.ts", "export const page = true;\n", false},
		{"pkg/shared/unrelated/code.go", "package unrelated\n", false},
		{"pkg/dev/new.go", "package dev\n", false},
		{"pkg/cli/new.go", "package cli\n", true},
		{"pkg/arm/new.go", "package arm\n", true},
		{"pkg/mac/new.go", "package mac\n", true},
		{"pkg/shared/new.go", "package shared\n", true},
		{"pkg/shared/assets/data.txt", "after\n", true},
		{"pkg/shared/assets/nested/new.txt", "new asset\n", true},
		{"build/app/new.go", "package main\n", true},
		{"Dockerfile.release", "FROM scratch\n", true},
		{"go.sum", "", true},
	}
	for _, change := range changes {
		writeFile(t, change.path, change.content)
		git(t, "add", change.path)
		git(t, "commit", "--quiet", "-m", change.path)
	}
	require.NoError(t, os.Remove("pkg/shared/deleted.go"))
	git(t, "add", "pkg/shared/deleted.go")
	git(t, "commit", "--quiet", "-m", "deleted dependency file")

	var output bytes.Buffer
	require.NoError(t, run("service.yaml", &output))
	var generated struct {
		Changelog struct {
			Use   string   `yaml:"use"`
			Paths []string `yaml:"paths"`
		} `yaml:"changelog"`
	}
	require.NoError(t, yaml.Unmarshal(output.Bytes(), &generated))
	require.Equal(t, "git", generated.Changelog.Use)
	require.Contains(t, generated.Changelog.Paths, "service.yaml")
	require.Contains(t, generated.Changelog.Paths, "base.yaml")
	require.Contains(t, generated.Changelog.Paths, "go.mod")
	args := append([]string{"log", "--format=%s", "app/v1.0.0..HEAD", "--"}, generated.Changelog.Paths...)
	history := strings.Split(strings.TrimSpace(git(t, args...)), "\n")
	expected := []string{"deleted dependency file"}
	for _, change := range changes {
		if change.include {
			expected = append(expected, change.path)
		}
	}
	require.ElementsMatch(t, expected, history)
}

func TestReleaseConfigFailsWithoutResolvableDependencies(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("GOWORK", "off")
	writeFile(t, "go.mod", "module example.test/release\n\ngo 1.26\n")
	writeFile(t, "release.yaml", `version: 2
builds:
  - main: ./missing
    goos: [linux]
    goarch: [amd64]
`)
	var output bytes.Buffer
	require.Error(t, run("release.yaml", &output))
	require.Empty(t, output.String())
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}

func git(t *testing.T, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", output)
	return string(output)
}
