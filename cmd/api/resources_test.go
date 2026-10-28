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
	require.ErrorContains(t, err, "jsno")
	require.Empty(t, output)
}

func TestMissingFlagsDoNotPrintToStdout(t *testing.T) {
	output, diagnostic, err := runCLI(t, "projects", "delete")
	require.ErrorContains(t, err, "project")
	require.Empty(t, output)
	require.Empty(t, diagnostic)
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
