package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunShowsHelpWithoutGeneratingOrUploading(t *testing.T) {
	var output bytes.Buffer
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("help must not upload")
		return nil, nil
	})}
	err := run(t.Context(), []string{"--help", "--file=missing.go", "--notion-page-id="}, &output, client)
	require.NoError(t, err)
	require.Empty(t, output.String())
}

func TestRunPreviewsStructCommentsAndTags(t *testing.T) {
	root := testRepository(t, map[string]string{
		"config.go": "package gateway\n\n// Config controls gateway listeners.\ntype Config struct {\n\t// Port serves customer traffic.\n\tPort int `toml:\"port\" config:\"default=7070,min=1,max=65535\"`\n\tImage string `toml:\"-\"`\n}\n",
	})
	var output bytes.Buffer
	err := run(t.Context(), []string{"--file", filepath.Join(root, "config.go")}, &output, http.DefaultClient)
	require.NoError(t, err)
	require.Equal(t, "<callout icon=\"ℹ️\" color=\"gray_bg\">\n\tGenerated from [Config](https://github.com/unkeyed/unkey/blob/main/config.go#L4). Edit the struct comments and tags in GitHub.\n</callout>\n\nConfig controls gateway listeners.\n\n# `port`\nType: `int` · Default: `7070` · Minimum: `1` · Maximum: `65535` · [Source](https://github.com/unkeyed/unkey/blob/main/config.go#L6) {color=\"gray\"}\nPort serves customer traffic.\n", output.String())
}

func TestRunSelectsTheStructFromTheExplicitFile(t *testing.T) {
	root := testRepository(t, map[string]string{
		"custom.go": "package gateway\n// GatewayConfig controls the customer listener.\ntype GatewayConfig struct {\n Port int `toml:\"port\"`\n}\n",
		"other.go":  "package gateway\ntype Other struct{}\n",
	})
	var output bytes.Buffer
	err := run(t.Context(), []string{"--file", filepath.Join(root, "custom.go"), "--struct", "GatewayConfig"}, &output, http.DefaultClient)
	require.NoError(t, err)
	require.Contains(t, output.String(), "GatewayConfig controls the customer listener.")
	require.Contains(t, output.String(), "blob/main/custom.go#L3")
	err = run(t.Context(), []string{"--file", filepath.Join(root, "other.go"), "--struct", "GatewayConfig"}, io.Discard, http.DefaultClient)
	require.ErrorContains(t, err, "struct GatewayConfig not found in")
}

func TestRunExpandsLocalAndSharedStructs(t *testing.T) {
	root := testRepository(t, map[string]string{
		"svc/frontline/config.go": "package frontline\nimport settings \"github.com/unkeyed/unkey/pkg/config\"\ntype Config struct {\n\tDatabase *settings.Database `toml:\"database\"`\n\tPrimary Redis `toml:\"primary\"`\n\tSecondary Redis `toml:\"secondary\"`\n}\ntype Redis struct {\n\tURL string `toml:\"url\"` // URL connects to Redis.\n}\n",
		"pkg/config/common.go":    "package config\n// Database holds connection settings.\ntype Database struct {\n\t// DSN must allow writes.\n\tDSN string `toml:\"dsn\" config:\"required,nonempty\"`\n}\ntype Unused struct {\n\tSecret string `toml:\"secret\"`\n}\n",
	})
	t.Chdir(root)
	var output bytes.Buffer
	err := run(t.Context(), []string{"--file", "svc/frontline/config.go", "--ref", "test-revision"}, &output, http.DefaultClient)
	require.NoError(t, err)
	headings := regexp.MustCompile("(?m)^#{1,4} `([^`]+)`$").FindAllStringSubmatch(output.String(), -1)
	var paths []string
	for _, heading := range headings {
		paths = append(paths, heading[1])
	}
	require.Equal(t, []string{"database", "database.dsn", "primary", "primary.url", "secondary", "secondary.url"}, paths)
	require.Contains(t, output.String(), "Database holds connection settings.")
	require.Contains(t, output.String(), "DSN must allow writes.")
	require.Contains(t, output.String(), "URL connects to Redis.")
	require.Contains(t, output.String(), "blob/test-revision/pkg/config/common.go#L5")
	require.NotContains(t, output.String(), root)
}

func TestRunNestsHeadingsByStructDepth(t *testing.T) {
	root := testRepository(t, map[string]string{
		"config.go": "package gateway\ntype Config struct {\n Control Control `toml:\"control\"`\n Literal string `toml:\"literal.key\"`\n Port int `toml:\"port\"`\n}\ntype Control struct {\n TLS *TLS `toml:\"tls\"`\n Token string `toml:\"token\"`\n}\ntype TLS struct {\n Client Client `toml:\"client\"`\n}\ntype Client struct {\n Certificate Certificate `toml:\"certificate\"`\n}\ntype Certificate struct {\n File string `toml:\"file\"`\n}\n",
	})
	var output bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"--file", filepath.Join(root, "config.go")}, &output, http.DefaultClient))
	headings := regexp.MustCompile("(?m)^#+ `[^`]+`$").FindAllString(output.String(), -1)
	require.Equal(t, []string{
		"# `control`",
		"## `control.tls`",
		"### `control.tls.client`",
		"#### `control.tls.client.certificate`",
		"#### `control.tls.client.certificate.file`",
		"## `control.token`",
		"# `literal.key`",
		"# `port`",
	}, headings)
}

func TestRunPreservesCommentCodeBlocksForNotion(t *testing.T) {
	root := testRepository(t, map[string]string{
		"config.go": "package gateway\n// Config accepts this example:\n//\n//\tport = 8181\n//\n// Keep this second paragraph.\ntype Config struct {\n Port int `toml:\"port\"`\n}\n",
	})
	var output bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"--file", filepath.Join(root, "config.go")}, &output, http.DefaultClient))
	require.Contains(t, output.String(), "Config accepts this example:\n\n```plaintext\nport = 8181\n```\n\nKeep this second paragraph.")
}

func TestRunLinksImportedFunctionsToSourceRevision(t *testing.T) {
	root := testRepository(t, map[string]string{
		"svc/gateway/config.go": "package gateway\nimport settings \"github.com/unkeyed/unkey/pkg/config\"\n// Config is loaded by [settings.Load].\ntype Config struct {\n Port int `toml:\"port\"`\n}\n",
		"pkg/config/load.go":    "package config\nfunc Load[T any](path string) (T, error) { panic(\"fixture\") }\n",
	})
	var output bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"--file", filepath.Join(root, "svc/gateway/config.go"), "--ref", "test-revision"}, &output, http.DefaultClient))
	require.Contains(t, output.String(), "Config is loaded by [settings.Load](https://github.com/unkeyed/unkey/blob/test-revision/pkg/config/load.go#L2).")
}

func TestRunLinksLocalSymbolsAcrossFiles(t *testing.T) {
	root := testRepository(t, map[string]string{
		"config.go":  "package gateway\n// Config uses [Config], [Config.Validate], [Config.Port], [DefaultPort], [ErrInvalid], and [New].\ntype Config struct {\n Port int `toml:\"port\"`\n}\n",
		"helpers.go": "package gateway\nconst DefaultPort = 7070\nvar ErrInvalid error\nfunc New() Config { return Config{} }\nfunc (c *Config) Validate() error { return nil }\n",
	})
	var output bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"--file", filepath.Join(root, "config.go")}, &output, http.DefaultClient))
	require.Contains(t, output.String(), "Config uses [Config](https://github.com/unkeyed/unkey/blob/main/config.go#L3), [Config.Validate](https://github.com/unkeyed/unkey/blob/main/helpers.go#L5), [Config.Port](https://github.com/unkeyed/unkey/blob/main/config.go#L4), [DefaultPort](https://github.com/unkeyed/unkey/blob/main/helpers.go#L2), [ErrInvalid](https://github.com/unkeyed/unkey/blob/main/helpers.go#L3), and [New](https://github.com/unkeyed/unkey/blob/main/helpers.go#L4).")
}

func TestRunResolvesInheritedCommentsInTheirDeclaringPackage(t *testing.T) {
	root := testRepository(t, map[string]string{
		"svc/gateway/config.go": "package gateway\nimport (\n \"github.com/unkeyed/unkey/pkg/shared\"\n helper \"github.com/unkeyed/unkey/pkg/wrong\"\n)\n// Config contains [shared.Config].\ntype Config struct {\n Shared shared.Config `toml:\"shared\"`\n}\nfunc (c *Config) Validate() error { return nil }\n",
		"pkg/shared/config.go":  "package shared\nimport helper \"github.com/unkeyed/unkey/pkg/right\"\n// Config calls [helper.Load] and [Config.Validate].\ntype Config struct {\n // Timeout uses [helper.Load].\n Timeout string `toml:\"timeout\"`\n}\nfunc (c *Config) Validate() error { return nil }\n",
		"pkg/right/load.go":     "package right\nfunc Load() {}\n",
		"pkg/wrong/load.go":     "package wrong\nfunc Load() {}\n",
	})
	var output bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"--file", filepath.Join(root, "svc/gateway/config.go")}, &output, http.DefaultClient))
	require.Contains(t, output.String(), "Config calls [helper.Load](https://github.com/unkeyed/unkey/blob/main/pkg/right/load.go#L2) and [Config.Validate](https://github.com/unkeyed/unkey/blob/main/pkg/shared/config.go#L8).")
	require.Contains(t, output.String(), "Timeout uses [helper.Load](https://github.com/unkeyed/unkey/blob/main/pkg/right/load.go#L2).")
	require.Contains(t, output.String(), "Config contains [shared.Config](https://github.com/unkeyed/unkey/blob/main/pkg/shared/config.go#L4).")
	require.NotContains(t, output.String(), "pkg/wrong")
}

func TestRunPreservesUnresolvedReferencesAndCodeExamples(t *testing.T) {
	root := testRepository(t, map[string]string{
		"config.go":          "package gateway\nimport settings \"github.com/unkeyed/unkey/pkg/config\"\n// Config keeps [settings.Missing], [Missing], and [storage.s3] as text.\n//\n// Use [guide].\n//\n//   - Call [settings.Load] with [time.Duration].\n//\n// For example:\n//\n//\t[settings.Load]\n//\n// [guide]: https://example.com/configuration\ntype Config struct {\n Port int `toml:\"port\"`\n}\n",
		"pkg/config/load.go": "package config\nfunc Load() {}\n",
	})
	var output bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"--file", filepath.Join(root, "config.go")}, &output, http.DefaultClient))
	require.Contains(t, output.String(), "Config keeps \\[settings.Missing], \\[Missing], and \\[storage.s3] as text.")
	require.Contains(t, output.String(), "[guide](https://example.com/configuration)")
	require.Contains(t, output.String(), "Call [settings.Load](https://github.com/unkeyed/unkey/blob/main/pkg/config/load.go#L2) with [time.Duration](https://pkg.go.dev/time#Duration).")
	require.Contains(t, output.String(), "```plaintext\n[settings.Load]\n```")
}

func TestRunLinksPackagesAndQualifiedSymbols(t *testing.T) {
	root := testRepository(t, map[string]string{
		"config.go":          "package gateway\nimport settings \"github.com/unkeyed/unkey/pkg/config\"\n// Config uses [settings], [github.com/unkeyed/unkey/pkg/config.Load], and [gateway.Config].\ntype Config struct {\n Port int `toml:\"port\"`\n}\n",
		"pkg/config/load.go": "package config\nfunc Load() {}\n",
	})
	var output bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"--file", filepath.Join(root, "config.go")}, &output, http.DefaultClient))
	require.Contains(t, output.String(), "Config uses [settings](https://github.com/unkeyed/unkey/tree/main/pkg/config), [github.com/unkeyed/unkey/pkg/config.Load](https://github.com/unkeyed/unkey/blob/main/pkg/config/load.go#L2), and [gateway.Config](https://github.com/unkeyed/unkey/blob/main/config.go#L4).")
}

func TestRunLeavesPlatformAmbiguousSymbolsUnlinked(t *testing.T) {
	root := testRepository(t, map[string]string{
		"config.go":      "package gateway\n// Config is loaded by [Load].\ntype Config struct {\n Port int `toml:\"port\"`\n}\n",
		"load_linux.go":  "package gateway\nfunc Load() {}\n",
		"load_darwin.go": "package gateway\nfunc Load() {}\n",
	})
	var output bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"--file", filepath.Join(root, "config.go")}, &output, http.DefaultClient))
	require.Contains(t, output.String(), "Config is loaded by \\[Load].")
}

func TestRunExpandsImportedStructSlices(t *testing.T) {
	root := testRepository(t, map[string]string{
		"svc/gateway/config.go": "package gateway\nimport settings \"github.com/unkeyed/unkey/pkg/cdc\"\ntype Watchers []settings.Config\ntype Config struct {\n Watchers Watchers `toml:\"watchers\"`\n}\n",
		"pkg/cdc/config.go":     "package cdc\ntype Config struct {\n Address string `toml:\"address\"` // Address selects the Vitess endpoint.\n}\n",
	})
	var output bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"--file", filepath.Join(root, "svc/gateway/config.go")}, &output, http.DefaultClient))
	require.Contains(t, output.String(), "## `watchers.address`\nType: `string` · [Source](https://github.com/unkeyed/unkey/blob/main/pkg/cdc/config.go#L3) {color=\"gray\"}\nAddress selects the Vitess endpoint.")
	require.Contains(t, output.String(), "blob/main/pkg/cdc/config.go#L3")
}

func TestRunUploadsMarkdownThenLocksExplicitPage(t *testing.T) {
	root := testRepository(t, map[string]string{
		"config.go": "package gateway\ntype Config struct {\n Port int `toml:\"port\"`\n}\n",
	})
	t.Setenv("NOTION_TOKEN", "test-token")
	var uploaded map[string]json.RawMessage
	var locked map[string]bool
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodPatch, req.Method)
		require.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))
		require.Equal(t, "2026-03-11", req.Header.Get("Notion-Version"))
		require.Equal(t, "application/json", req.Header.Get("Content-Type"))
		if req.URL.String() == "https://api.notion.com/v1/pages/3ef512d643f3815f8442c1bcf3bb1562" {
			require.NotEmpty(t, uploaded)
			require.NoError(t, json.NewDecoder(req.Body).Decode(&locked))
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(notionJSON(t, map[string]any{"object": "page", "is_locked": true})))}, nil
		}
		require.Equal(t, "https://api.notion.com/v1/pages/3ef512d643f3815f8442c1bcf3bb1562/markdown", req.URL.String())
		require.NoError(t, json.NewDecoder(req.Body).Decode(&uploaded))
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(notionJSON(t, map[string]any{"object": "page_markdown"})))}, nil
	})}
	err := run(t.Context(), []string{"--file", filepath.Join(root, "config.go"), "--notion-page-id", "3ef512d6-43f3-815f-8442-c1bcf3bb1562"}, io.Discard, client)
	require.NoError(t, err)
	require.Len(t, uploaded, 2)
	require.JSONEq(t, `"replace_content"`, string(uploaded["type"]))
	var replacement struct {
		Markdown      string `json:"new_str"`
		AllowDeleting bool   `json:"allow_deleting_content"`
	}
	require.NoError(t, json.Unmarshal(uploaded["replace_content"], &replacement))
	require.False(t, replacement.AllowDeleting)
	require.Contains(t, replacement.Markdown, "# `port`\nType: `int`")
	require.Equal(t, map[string]bool{"is_locked": true}, locked)
}

func TestRunRejectsInvalidUploadInputsBeforeRequests(t *testing.T) {
	root := testRepository(t, map[string]string{
		"config.go": "package gateway\ntype Config struct {\n Port int `toml:\"port\"`\n}\n",
	})
	for _, tc := range []struct {
		name, pageID, token, message string
	}{
		{"missing token", "3ef512d643f3815f8442c1bcf3bb1562", "", "NOTION_TOKEN is required"},
		{"invalid page", "wrong/page", "test-token", "notion-page-id must contain 32 hexadecimal characters"},
		{"empty explicit page", "", "test-token", "notion-page-id must contain 32 hexadecimal characters"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("NOTION_TOKEN", tc.token)
			requests := 0
			client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				requests++
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(notionJSON(t, map[string]any{"object": "page_markdown"})))}, nil
			})}
			err := run(t.Context(), []string{"--file", filepath.Join(root, "config.go"), "--notion-page-id=" + tc.pageID}, io.Discard, client)
			require.ErrorContains(t, err, tc.message)
			require.Zero(t, requests)
		})
	}
}

func TestRunReportsUploadFailuresWithoutCreatingPages(t *testing.T) {
	root := testRepository(t, map[string]string{
		"config.go": "package gateway\ntype Config struct {\n Port int `toml:\"port\"`\n}\n",
	})
	t.Setenv("NOTION_TOKEN", "test-token")
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"missing page", http.StatusNotFound, notionJSON(t, map[string]any{"object": "error"})},
		{"child deletion refused", http.StatusBadRequest, notionJSON(t, map[string]any{"object": "error"})},
		{"queued is not completed", http.StatusAccepted, notionJSON(t, map[string]any{"object": "async_task"})},
		{"malformed response", http.StatusOK, `{`},
		{"unexpected object", http.StatusOK, notionJSON(t, map[string]any{"object": "error"})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requests := 0
			client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				require.Equal(t, http.MethodPatch, req.Method)
				return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body))}, nil
			})}
			err := run(t.Context(), []string{"--file", filepath.Join(root, "config.go"), "--notion-page-id=3ef512d643f3815f8442c1bcf3bb1562"}, io.Discard, client)
			require.Error(t, err)
			require.NotContains(t, err.Error(), "test-token")
			require.Equal(t, 1, requests)
		})
	}
}

func TestRunReportsLockFailureAfterUploading(t *testing.T) {
	root := testRepository(t, map[string]string{
		"config.go": "package gateway\ntype Config struct {\n Port int `toml:\"port\"`\n}\n",
	})
	t.Setenv("NOTION_TOKEN", "test-token")
	for _, tc := range []struct {
		name    string
		status  int
		body    string
		message string
	}{
		{"permission denied", http.StatusForbidden, notionJSON(t, map[string]any{"object": "error"}), "HTTP 403"},
		{"still unlocked", http.StatusOK, notionJSON(t, map[string]any{"object": "page", "is_locked": false}), "Notion did not confirm the page lock"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requests := 0
			client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				if strings.HasSuffix(req.URL.Path, "/markdown") {
					return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(notionJSON(t, map[string]any{"object": "page_markdown"})))}, nil
				}
				return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body))}, nil
			})}
			err := run(t.Context(), []string{"--file", filepath.Join(root, "config.go"), "--notion-page-id=3ef512d643f3815f8442c1bcf3bb1562"}, io.Discard, client)
			require.ErrorContains(t, err, "content uploaded but page lock failed")
			require.ErrorContains(t, err, tc.message)
			require.NotContains(t, err.Error(), "test-token")
			require.Equal(t, 2, requests)
		})
	}
}

func notionJSON(t *testing.T, value map[string]any) string {
	t.Helper()
	body, err := json.Marshal(value)
	require.NoError(t, err)
	return string(body)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func testRepository(t *testing.T, sources map[string]string) string {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module github.com/unkeyed/unkey\n"), 0o600))
	for name, source := range sources {
		filename := filepath.Join(root, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(filename), 0o700))
		require.NoError(t, os.WriteFile(filename, []byte(source), 0o600))
	}
	return root
}
