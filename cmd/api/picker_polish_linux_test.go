package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v3/models/apierrors"
	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestPickerNarrowTerminalKeepsHintsCountsAndDistinctIDs(t *testing.T) {
	projects := make([]components.Project, 6)
	for i := range projects {
		projects[i] = components.Project{ID: uid.New(uid.ProjectPrefix), Name: "Fictional similarly named project", Slug: fmt.Sprintf("fixture-%d", i)}
	}
	requests := make(chan string, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/v2/projects.listProjects" {
			t.Errorf("unexpected endpoint %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if err := json.NewEncoder(w).Encode(components.V2ProjectsListProjectsResponseBody{
			Data: projects, Pagination: components.Pagination{HasMore: true, Cursor: new(uid.New(uid.TestPrefix))},
		}); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	p := startPickerPTY(t, []string{"projects", "delete", "--root-key=" + uid.New(uid.KeyPrefix), "--api-url=" + server.URL}, 30, 14)
	p.expect("of 6 resources")
	screen := p.screen()
	require.Contains(t, screen, "Filter or enter a value")
	require.Contains(t, screen, "↑/↓ move · Enter select")
	require.Contains(t, screen, "Ctrl+C to cancel")
	for _, project := range projects[:3] {
		require.Contains(t, screen, project.ID)
	}
	recordGuided(t, "picker-narrow-30x14", p.screen(), "")
	p.send(strings.Repeat("\x1b[B", len(projects)))
	p.expect("❯ Next page")
	screen = p.screen()
	require.Contains(t, screen, "of 6 resources")
	require.NotContains(t, screen, "7 resources")
	require.Contains(t, screen, "↑/↓ move · Enter select")
	require.Contains(t, screen, "Ctrl+C to cancel")
	p.send("absent")
	p.expect("Search all for: absent")
	screen = p.screen()
	require.Contains(t, screen, "0 resources")
	require.Contains(t, screen, "Use: absent")
	require.Contains(t, screen, "Search all for: absent")
	require.Contains(t, screen, "↑/↓ move · Enter select")
	require.Contains(t, screen, "Ctrl+C to cancel")
	recordGuided(t, "picker-narrow-actions-not-resources", p.screen(), "")
	p.send("\x03")
	require.Error(t, p.finish())
	require.Len(t, requests, 1)
}

func TestPickerNamesIncludeProjectSlugAndKeyStartWithoutSecrets(t *testing.T) {
	for _, resource := range []string{"projects", "root-keys"} {
		t.Run(resource, func(t *testing.T) {
			id, prefix := uid.New(uid.ProjectPrefix), "fictional_start_"
			secret := "fictional-never-show-key-end"
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				var response any
				switch r.URL.Path {
				case "/v2/projects.listProjects":
					response = components.V2ProjectsListProjectsResponseBody{Data: []components.Project{{ID: id, Name: "Fictional project", Slug: "fictional-production"}}}
				case "/v2/rootKeys.listKeys":
					response = components.V2RootKeysListKeysResponseBody{Data: []components.V2RootKeysListKeysResponseData{{KeyID: uid.New(uid.KeyPrefix), Name: new("Fictional CI"), Start: prefix, End: secret}}}
				default:
					t.Errorf("unexpected endpoint %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(server.Close)
			p := startPickerPTY(t, []string{resource, "delete", "--root-key=" + uid.New(uid.KeyPrefix), "--api-url=" + server.URL}, 100)
			if resource == "projects" {
				p.expect("Fictional project · fictional-production")
			} else {
				p.expect("Fictional CI · " + prefix + "…")
			}
			require.NotContains(t, p.transcript.String(), secret)
			recordGuided(t, "picker-label-"+resource, p.screen(), "")
			p.send("\x03")
			require.Error(t, p.finish())
		})
	}
}

func TestPickerEnvironmentVariableNamesUseScopeAndNeverShowValues(t *testing.T) {
	for _, mode := range []string{"select", "denied", "cancel", "edit"} {
		t.Run(mode, func(t *testing.T) {
			project, app, environment := "fictional-project", "fictional-app", "fictional-preview"
			secret := "fictional-value-never-render"
			lists := make(chan components.V2EnvironmentsListEnvironmentVariablesRequestBody, 4)
			mutations := make(chan components.V2EnvironmentsRemoveEnvironmentVariablesRequestBody, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/v2/environments.listEnvironmentVariables":
					var request components.V2EnvironmentsListEnvironmentVariablesRequestBody
					if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
						t.Error(err)
						return
					}
					lists <- request
					if mode == "denied" {
						w.WriteHeader(http.StatusForbidden)
						if err := json.NewEncoder(w).Encode(apierrors.ForbiddenErrorResponse{Error_: components.BaseError{Status: http.StatusForbidden, Detail: "Fixture permission denied"}}); err != nil {
							t.Error(err)
						}
						return
					}
					if err := json.NewEncoder(w).Encode(components.V2EnvironmentsListEnvironmentVariablesResponseBody{Data: []components.EnvironmentVariable{
						{Key: "CACHE_HOST", Kind: components.EnvironmentVariableKindRecoverable, Value: &secret, Description: &secret},
						{Key: "LOG_LEVEL", Kind: components.EnvironmentVariableKindWriteonly},
					}}); err != nil {
						t.Error(err)
					}
				case "/v2/environments.removeEnvironmentVariables":
					var request components.V2EnvironmentsRemoveEnvironmentVariablesRequestBody
					if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
						t.Error(err)
						return
					}
					mutations <- request
					if err := json.NewEncoder(w).Encode(components.V2EnvironmentsRemoveEnvironmentVariablesResponseBody{}); err != nil {
						t.Error(err)
					}
				default:
					t.Errorf("unexpected endpoint %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			t.Cleanup(server.Close)
			args := []string{"environments", "variables", "remove", "--project=" + project, "--app=" + app, "--environment=" + environment, "--root-key=" + uid.New(uid.KeyPrefix), "--api-url=" + server.URL}
			if mode == "edit" {
				args = append(args, "--variables=KEEP_EXISTING,LOG_LEVEL", "--interactive")
			}
			columns, rows := uint16(100), uint16(24)
			if mode == "select" {
				columns, rows = 30, 14
			}
			p := startPickerPTY(t, args, columns, rows)
			if mode == "edit" {
				p.expect("❯ Cancel")
				p.send("\x1b[A\r")
				p.expect("Change Variables")
				p.send("\x1b[H\x1b[B\r")
				p.expect("Keep this list")
				p.send("\x1b[F\r")
				p.expect("Change value")
				p.send("\r")
				p.expect("❯ LOG_LEVEL")
				recordGuided(t, "picker-variable-edit-selection", p.screen(), "")
				p.send("\x1b[A\r")
				p.expect("Keep this list")
				p.send("\r")
			} else {
				p.expect("Add item")
				require.Contains(t, p.screen(), "↑/↓ move · Enter select")
				require.Contains(t, p.screen(), "Ctrl+C to cancel")
				recordGuided(t, "picker-array-controls-"+mode, p.screen(), "")
				p.send("\r")
				if mode == "denied" {
					p.expect("Enter a value manually instead")
					p.send("MANUAL_NAME\r")
				} else {
					p.expect("CACHE_HOST")
					require.Contains(t, p.screen(), "? Variables")
					require.Contains(t, p.screen(), "LOG_LEVEL")
					require.Contains(t, p.screen(), "2 resources")
					require.Contains(t, p.screen(), "↑/↓ move · Enter select")
					require.Contains(t, p.screen(), "Ctrl+C to cancel")
					require.NotContains(t, p.transcript.String(), secret)
					recordGuided(t, "picker-variable-names-"+mode, p.screen(), "")
					if mode == "cancel" {
						p.send("\x03")
						require.Error(t, p.finish())
						require.Empty(t, mutations)
						return
					}
					p.send("\x1b[B\r")
				}
				p.expect("Use this list (1 item)")
				p.send("\x1b[B\r")
			}
			p.expect("❯ Cancel")
			require.Empty(t, mutations)
			p.send("\x1b[B\r")
			require.NoError(t, p.finish(), p.transcript.String())
			require.NotContains(t, p.transcript.String(), secret)
			require.Len(t, lists, 1)
			request := <-lists
			require.Equal(t, project, request.Project)
			require.Equal(t, app, request.App)
			require.Equal(t, environment, request.Environment)
			require.Nil(t, request.Cursor)
			require.Len(t, mutations, 1)
			removed := <-mutations
			require.Equal(t, project, removed.Project)
			require.Equal(t, app, removed.App)
			require.Equal(t, environment, removed.Environment)
			expected := []string{"LOG_LEVEL"}
			if mode == "denied" {
				expected = []string{"MANUAL_NAME"}
			} else if mode == "edit" {
				expected = []string{"KEEP_EXISTING", "CACHE_HOST"}
			}
			require.Equal(t, expected, removed.Variables)
		})
	}
}

func TestPickerInvalidManualValueRemainsEditableWithoutRefetch(t *testing.T) {
	id := uid.New(uid.ProjectPrefix)
	lists := make(chan string, 4)
	requests := make(chan components.V2ProjectsGetProjectRequestBody, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/projects.listProjects":
			lists <- r.URL.Path
			if err := json.NewEncoder(w).Encode(components.V2ProjectsListProjectsResponseBody{Data: []components.Project{{ID: id, Name: "Fictional project"}}}); err != nil {
				t.Error(err)
			}
		case "/v2/projects.getProject":
			var request components.V2ProjectsGetProjectRequestBody
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Error(err)
				return
			}
			requests <- request
			if err := json.NewEncoder(w).Encode(components.V2ProjectsGetProjectResponseBody{Data: components.Project{ID: id, Name: "Fictional project"}}); err != nil {
				t.Error(err)
			}
		default:
			t.Errorf("unexpected endpoint %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	p := startPickerPTY(t, []string{"projects", "get", "--root-key=" + uid.New(uid.KeyPrefix), "--api-url=" + server.URL}, 30, 14)
	p.expect(id)
	p.send("zz\r")
	p.expect("Enter at least 3 characters")
	require.Contains(t, p.screen(), "  zz\n")
	require.Contains(t, p.screen(), "Use: zz")
	require.Contains(t, p.screen(), "↑/↓ move · Enter select")
	require.Contains(t, p.screen(), "Ctrl+C to cancel")
	require.Empty(t, requests)
	recordGuided(t, "picker-invalid-manual-retained", p.screen(), "")
	p.send("-fixture\r")
	require.NoError(t, p.finish(), p.transcript.String())
	require.Len(t, lists, 1)
	require.Len(t, requests, 1)
	require.Equal(t, "zz-fixture", (<-requests).Project)
}
