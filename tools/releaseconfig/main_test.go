package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestReleaseConfigScopesHistoryToReleaseDependencies(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("GOWORK", "off")
	t.Setenv("GORELEASER_PREVIOUS_TAG", "")
	writeFile(t, "go.mod", "module example.test/release\n\ngo 1.26\n")
	writeFile(t, "service.yaml", `version: 2
project_name: app
monorepo:
  tag_prefix: app/
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
changelog:
  filters:
    exclude: ['^test:']
`)
	writeFile(t, "build/app/main.go", `package main
import "example.test/release/pkg/shared"
func main() { shared.Run() }
`)
	writeFile(t, "build/app/release.go", "//go:build cli_release\n\npackage main\nimport _ \"example.test/release/pkg/cli\"\n")
	writeFile(t, "build/app/dev.go", "//go:build !cli_release\n\npackage main\nimport _ \"example.test/release/pkg/dev\"\n")
	writeFile(t, "pkg/shared/shared.go", `package shared
import (
 "embed"
 "reflect"
 _ "example.test/release/pkg/unusedassets"
)
//go:embed	assets
var assets embed.FS
func Run() {
 data, err := assets.ReadFile("assets/data.txt")
 println(string(data), err, Used(), Generic(2), Catalog[0], Platform(), Arch(), linkCaller())
 var value interface{ Value() int } = &Item{}
 println(value.Value())
 reflect.ValueOf(ReflectItem{}).MethodByName("Text").Call(nil)
}
`)
	writeFile(t, "pkg/shared/used.go", "package shared\nfunc Used() int { return 1 }\n")
	writeFile(t, "pkg/shared/dead.go", "package shared\nfunc Dead() int { return 1 }\n")
	writeFile(t, "pkg/shared/generic.go", "package shared\nfunc Generic[T ~int](value T) T { return value+1 }\n")
	aliased := `package shared
import _ "unsafe"
//go:linkname LinkTarget example.test/release/pkg/shared.LinkTargetAlias
func LinkTarget() int { return 1 }
//go:linkname linkCaller example.test/release/pkg/shared.LinkTargetAlias
func linkCaller() int
`
	writeFile(t, "pkg/shared/aliased.go", aliased)
	writeFile(t, "pkg/shared/method.go", "package shared\ntype Item struct{}\nfunc (*Item) Value() int { return 1 }\n")
	writeFile(t, "pkg/shared/method_unused.go", "package shared\nfunc (*Item) unused() int { return 1 }\n")
	writeFile(t, "pkg/shared/reflected.go", "package shared\ntype ReflectItem struct{}\nfunc (ReflectItem) Text() string { return \"before\" }\n")
	writeFile(t, "pkg/shared/data.go", "package shared\nvar Catalog = []int{1}\n")
	writeFile(t, "pkg/shared/comment.go", "package shared\nfunc Comment() {}\n")
	writeFile(t, "pkg/shared/effect.go", "package shared\nfunc effect(value int) int { println(value); return value }\n")
	writeFile(t, "pkg/shared/order.go", "package shared\nvar first = effect(1)\nvar second = effect(2)\n")
	writeFile(t, "pkg/shared/blank.go", "package shared\nvar _ = effect(1)\nvar _ = effect(2)\n")
	writeFile(t, "pkg/shared/arch_arm64.go", "package shared\nimport _ \"example.test/release/pkg/arm\"\nfunc Arch() int { return 1 }\n")
	writeFile(t, "pkg/shared/arch_amd64.go", "package shared\nfunc Arch() int { return 2 }\n")
	writeFile(t, "pkg/shared/os_darwin.go", "package shared\nimport _ \"example.test/release/pkg/mac\"\nfunc Platform() int { return 1 }\n")
	writeFile(t, "pkg/shared/os_linux.go", "package shared\nfunc Platform() int { return 2 }\n")
	writeFile(t, "pkg/unusedassets/assets.go", "package unusedassets\nimport _ \"embed\"\n//go:embed data.txt\nvar unused string\n")
	writeFile(t, "pkg/unusedassets/data.txt", "before\n")
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
		{"pkg/cli/new.go", "package cli\nfunc init() { println(1) }\n", true},
		{"pkg/arm/new.go", "package arm\nfunc init() { println(2) }\n", true},
		{"pkg/mac/new.go", "package mac\nfunc init() { println(3) }\n", true},
		{"pkg/shared/new.go", "package shared\nconst Limit = 42\n", true},
		{"pkg/shared/used.go", "package shared\nfunc Used() int { return 7 }\n", true},
		{"pkg/shared/dead.go", "package shared\nfunc Dead() int { return 9 }\n", false},
		{"pkg/shared/dead_new.go", "package shared\nfunc UnusedNew() int { return 11 }\n", false},
		{"pkg/shared/aliased.go", strings.Replace(aliased, "return 1", "return 9", 1), true},
		{"pkg/shared/generic.go", "package shared\nfunc Generic[T ~int](value T) T { return value+5 }\n", true},
		{"pkg/shared/method.go", "package shared\ntype Item struct{}\nfunc (*Item) Value() int { return 12 }\n", true},
		{"pkg/shared/method_unused.go", "package shared\nfunc (*Item) unused() int { return 13 }\n", false},
		{"pkg/shared/reflected.go", "package shared\ntype ReflectItem struct{}\nfunc (ReflectItem) Text() string { return \"after\" }\n", true},
		{"pkg/shared/arch_arm64.go", "package shared\nimport _ \"example.test/release/pkg/arm\"\nfunc Arch() int { return 7 }\n", true},
		{"pkg/shared/os_linux.go", "package shared\nfunc Platform() int { return 8 }\n", true},
		{"pkg/shared/data.go", "package shared\nvar Catalog = []int{13}\n", true},
		{"pkg/shared/comment.go", "package shared\n// Documentation only.\nfunc Comment() {}\n", false},
		{"pkg/shared/order.go", "package shared\nvar second = effect(2)\nvar first = effect(1)\n", true},
		{"pkg/shared/blank.go", "package shared\nvar _ = effect(5)\nvar _ = effect(2)\n", true},
		{"pkg/shared/shared_test.go", "package shared\n", false},
		{"pkg/shared/README.md", "Package documentation\n", false},
		{"pkg/shared/assets/data.txt", "after\n", true},
		{"pkg/shared/assets/nested/new.txt", "new asset\n", true},
		{"pkg/unusedassets/data.txt", "after\n", false},
		{"build/app/new.go", "package main\nfunc init() { println(4) }\n", true},
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
	writeFile(t, "pkg/shared/used.go", "package shared\nfunc Used() int { return 19 }\n")
	git(t, "add", "pkg/shared/used.go")
	git(t, "commit", "--quiet", "-m", "same subject [.*]")
	writeFile(t, "pkg/shared/dead.go", "package shared\nfunc Dead() int { return 17 }\n")
	git(t, "add", "pkg/shared/dead.go")
	git(t, "commit", "--quiet", "-m", "same subject [.*]")
	git(t, "tag", "other/v9.0.0")
	git(t, "tag", "app/v1.1.0")

	var output bytes.Buffer
	require.NoError(t, run("service.yaml", &output))
	var generated struct {
		Changelog struct {
			Use     string   `yaml:"use"`
			Paths   []string `yaml:"paths"`
			Format  string   `yaml:"format"`
			Filters struct {
				Exclude []string `yaml:"exclude"`
			} `yaml:"filters"`
		} `yaml:"changelog"`
	}
	require.NoError(t, yaml.Unmarshal(output.Bytes(), &generated))
	require.Equal(t, "git", generated.Changelog.Use)
	require.Contains(t, generated.Changelog.Paths, "service.yaml")
	require.Contains(t, generated.Changelog.Paths, "base.yaml")
	require.Contains(t, generated.Changelog.Paths, "go.mod")
	require.Contains(t, generated.Changelog.Filters.Exclude, "^test:")
	args := append([]string{"log", "--format=%H%x00%s", "app/v1.0.0..HEAD", "--"}, generated.Changelog.Paths...)
	format, err := template.New("changelog").Parse(generated.Changelog.Format)
	require.NoError(t, err)
	var history []string
	for _, line := range strings.Split(strings.TrimSpace(git(t, args...)), "\n") {
		sha, subject, ok := strings.Cut(line, "\x00")
		require.True(t, ok)
		excluded := false
		for _, pattern := range generated.Changelog.Filters.Exclude {
			matched, err := regexp.MatchString(pattern, subject)
			require.NoError(t, err)
			excluded = excluded || matched
		}
		if excluded {
			continue
		}
		history = append(history, subject)
		var rendered bytes.Buffer
		require.NoError(t, format.Execute(&rendered, struct{ SHA, Message string }{sha, subject}))
		require.NotContains(t, rendered.String(), "outside analyzed range")
		switch subject {
		case "pkg/shared/used.go":
			require.Contains(t, rendered.String(), "linked `example.test/release/pkg/shared.Used`")
		case "pkg/shared/generic.go":
			require.Contains(t, rendered.String(), "linked `example.test/release/pkg/shared.Generic`")
		case "pkg/shared/method.go":
			require.Contains(t, rendered.String(), "linked `example.test/release/pkg/shared.(*Item).Value`")
		case "pkg/shared/data.go":
			require.Contains(t, rendered.String(), "linked `example.test/release/pkg/shared.Catalog`")
		case "pkg/shared/reflected.go":
			require.Contains(t, rendered.String(), "linked `example.test/release/pkg/shared.ReflectItem.Text`")
		}
	}
	expected := []string{"deleted dependency file", "same subject [.*]", "same subject [.*]"}
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
