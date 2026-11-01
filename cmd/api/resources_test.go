package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestResourceCommandsCoverLegacyEndpoints(t *testing.T) {
	t.Setenv("UNKEY_OUTPUT", "")
	paths := map[string]bool{}
	for _, group := range Cmd().Commands {
		for _, command := range group.Commands {
			legacy := group.Name + " " + command.Name
			path, ok := resourcePaths[legacy]
			require.True(t, ok, "add a resource path for %s", legacy)
			require.False(t, paths[path], "duplicate resource path: %s", path)
			paths[path] = true
			for _, invocation := range []string{path, "api " + legacy} {
				t.Run(invocation, func(t *testing.T) {
					output, _, err := runCLI(t, append(strings.Fields(invocation), "--help")...)
					require.NoError(t, err)
					require.Contains(t, output, "unkey "+invocation+" [options]")
					require.NotContains(t, output, "[arguments...]")
					if invocation == path {
						require.NotContains(t, output, "unkey api ")
					}
				})
			}
		}
	}
	require.Len(t, paths, len(resourcePaths), "remove paths for deleted endpoints")
}

func TestProjectsListHumanAndMachineOutput(t *testing.T) {
	t.Setenv("UNKEY_OUTPUT", "")
	requestID := uid.New(uid.RequestPrefix)
	projectID := uid.New(uid.ProjectPrefix)
	response := components.V2ProjectsListProjectsResponseBody{
		Meta:       components.Meta{RequestID: requestID},
		Data:       []components.Project{{ID: projectID, Name: "Payments", Slug: "payments", CreatedAt: 1700000000000, UpdatedAt: new(int64(1700003600000))}},
		Pagination: components.Pagination{Cursor: new("next'page"), HasMore: true},
	}
	var requests []components.V2ProjectsListProjectsRequestBody
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/projects.listProjects" {
			t.Errorf("unexpected endpoint: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var request components.V2ProjectsListProjectsRequestBody
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		requests = append(requests, request)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	flags := []string{"--root-key=" + uid.New(uid.KeyPrefix), "--api-url=" + server.URL, "--search=pay", "--limit=2"}
	for _, path := range []string{"projects list", "api projects list-projects"} {
		t.Run(path, func(t *testing.T) {
			args := append(strings.Fields(path), flags...)
			output, diagnostic, err := runCLI(t, args...)
			require.NoError(t, err)
			require.Empty(t, diagnostic)
			header, _, _ := strings.Cut(output, "\n")
			require.Equal(t, []string{"NAME", "SLUG", "UPDATED"}, strings.Fields(header))
			require.NotContains(t, output, projectID)
			require.Contains(t, output, "2023-11-14T23:13:20Z")
			require.Contains(t, output, "Payments")
			require.NotContains(t, output, requestID)
			require.Contains(t, output, `--cursor='next'"'"'page'`)
			for _, format := range []string{"--json", "--output=json"} {
				output, diagnostic, err = runCLI(t, append(args, format, "--debug")...)
				require.NoError(t, err)
				var got components.V2ProjectsListProjectsResponseBody
				require.NoError(t, json.Unmarshal([]byte(output), &got))
				require.Equal(t, response, got)
				require.Equal(t, "Request ID: "+requestID+"\n", diagnostic)
			}
		})
	}
	require.Len(t, requests, 6)
	for _, request := range requests {
		require.Equal(t, int64(2), *request.Limit)
		require.Equal(t, "pay", *request.Search)
	}

	body, err := json.Marshal(components.V2ProjectsListProjectsRequestBody{Limit: new(int64(2)), Search: new("pay")})
	require.NoError(t, err)
	output, diagnostic, err := runCLI(t, "projects", "list", "--body="+string(body), flags[0], flags[1])
	require.NoError(t, err)
	require.Empty(t, diagnostic)
	require.Contains(t, output, `In --body, set cursor to "next'page"`)
	require.NotContains(t, output, "--cursor=")

	response.Data = []components.Project{}
	response.Pagination = components.Pagination{}
	output, diagnostic, err = runCLI(t, append([]string{"projects", "list"}, flags...)...)
	require.NoError(t, err)
	require.Equal(t, "No projects found.\n", output)
	require.Empty(t, diagnostic)
}

func TestInvalidOutputDoesNotSendMutation(t *testing.T) {
	t.Setenv("UNKEY_OUTPUT", "")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("invalid output must fail before a request is sent")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)
	output, _, err := runCLI(t, "projects", "delete", "--project=payments", "--output=jsno", "--root-key="+uid.New(uid.KeyPrefix), "--api-url="+server.URL)
	require.ErrorContains(t, err, "invalid value for flag --output")
	require.Empty(t, output)
}

func TestMissingFlagsDoNotPrintToStdout(t *testing.T) {
	output, diagnostic, err := runCLI(t, "projects", "delete")
	require.ErrorContains(t, err, "project")
	require.Empty(t, output)
	require.Empty(t, diagnostic)
}

func TestWorkspaceFlagUpdateRequiresEnabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("missing --enabled must fail before a request is sent")
		w.WriteHeader(http.StatusBadRequest)
	}))
	t.Cleanup(server.Close)
	for _, path := range []string{"workspace flags update", "api workspace update-flag"} {
		t.Run(path, func(t *testing.T) {
			args := append(strings.Fields(path), "--slug=example_feature", "--root-key="+uid.New(uid.KeyPrefix), "--api-url="+server.URL)
			output, diagnostic, err := runCLI(t, args...)
			require.ErrorContains(t, err, "required flag missing: enabled")
			require.Empty(t, output)
			require.Empty(t, diagnostic)
		})
	}
}

func TestBodylessWorkspaceEndpoints(t *testing.T) {
	for _, test := range []struct {
		legacy   string
		endpoint string
		response any
	}{
		{"workspace get-flags", "workspace.getFlags", components.V2WorkspaceGetFlagsResponseBody{Data: map[string]components.V2WorkspaceGetFlagsFlag{"example_feature": {Enabled: false, Description: "Example feature", AllowOptIn: true}}}},
		{"workspace get-limits", "workspace.getLimits", components.V2WorkspaceGetLimitsResponseBody{}},
	} {
		t.Run(test.legacy, func(t *testing.T) {
			key := uid.New(uid.KeyPrefix)
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != http.MethodPost || r.URL.Path != "/v2/"+test.endpoint || r.Header.Get("Authorization") != "Bearer "+key {
					t.Error("unexpected request method, route, or authorization")
				}
				body, err := io.ReadAll(r.Body)
				if err != nil || len(body) != 0 {
					t.Errorf("expected an empty request body, got %q: %v", body, err)
				}
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(test.response); err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(server.Close)
			want, err := json.Marshal(test.response)
			require.NoError(t, err)
			for _, path := range []string{resourcePaths[test.legacy], "api " + test.legacy} {
				args := append(strings.Fields(path), "--root-key="+key, "--api-url="+server.URL, "--json")
				output, diagnostic, err := runCLI(t, args...)
				require.NoError(t, err)
				require.Empty(t, diagnostic)
				require.JSONEq(t, string(want), output)
				_, _, err = runCLI(t, append(args, "--body="+string(want))...)
				require.ErrorContains(t, err, "body")
			}
			require.Equal(t, 2, calls)
		})
	}
}

func TestRootKeyCreationSendsBearerAndReturnsSecret(t *testing.T) {
	key := uid.New(uid.KeyPrefix)
	response := components.V2RootKeysCreateKeyResponseBody{
		Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)},
		Data: components.V2RootKeysCreateKeyResponseData{KeyID: uid.New(uid.KeyPrefix), Key: uid.New(uid.KeyPrefix)},
	}
	permission := "unkey:v1:" + uid.New(uid.WorkspacePrefix) + ":rootKeys/*#read"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/rootKeys.createKey" || r.Header.Get("Authorization") != "Bearer "+key {
			t.Error("unexpected route or authorization")
		}
		var request components.V2RootKeysCreateKeyRequestBody
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		if len(request.Permissions) != 1 || request.Permissions[0] != permission {
			t.Error("incorrect permissions")
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	for _, path := range []string{"root-keys create", "api root-keys create-key"} {
		args := append(strings.Fields(path), "--root-key="+key, "--api-url="+server.URL, "--permissions="+permission)
		output, diagnostic, err := runCLI(t, args...)
		require.NoError(t, err)
		require.Empty(t, diagnostic)
		require.Equal(t, 1, strings.Count(output, response.Data.Key))
		require.Contains(t, output, response.Data.KeyID)
		require.NotContains(t, output, response.Meta.RequestID)
		output, diagnostic, err = runCLI(t, append(args, "--json")...)
		require.NoError(t, err)
		require.Empty(t, diagnostic)
		var got components.V2RootKeysCreateKeyResponseBody
		require.NoError(t, json.Unmarshal([]byte(output), &got))
		require.Equal(t, response, got)
	}
}

func runCLI(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	stdout := captureFile(t)
	stderr := captureFile(t)
	originalOut, originalErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = stdout, stderr
	t.Cleanup(func() { os.Stdout, os.Stderr = originalOut, originalErr })
	root := &cli.Command{Name: "unkey", Commands: append(ResourceCommands(), Cmd())}
	err := root.Run(context.Background(), append([]string{"unkey"}, args...))
	os.Stdout, os.Stderr = originalOut, originalErr
	require.NoError(t, stdout.Sync())
	require.NoError(t, stderr.Sync())
	_, seekErr := stdout.Seek(0, io.SeekStart)
	require.NoError(t, seekErr)
	_, seekErr = stderr.Seek(0, io.SeekStart)
	require.NoError(t, seekErr)
	out, readErr := io.ReadAll(stdout)
	require.NoError(t, readErr)
	diagnostic, readErr := io.ReadAll(stderr)
	require.NoError(t, readErr)
	return string(out), string(diagnostic), err
}

func captureFile(t *testing.T) *os.File {
	t.Helper()
	file, err := os.Create(filepath.Join(t.TempDir(), "output"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, file.Close()) })
	return file
}
