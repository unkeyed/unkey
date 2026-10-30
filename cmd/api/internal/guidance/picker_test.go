package guidance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v3/models/apierrors"
	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/prompt"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestResourcePickerSearchResetsCursorAndPagingIsExplicit(t *testing.T) {
	firstID, secondID, foundID := uid.New(uid.ProjectPrefix), uid.New(uid.ProjectPrefix), uid.New(uid.ProjectPrefix)
	cursor := uid.New(uid.TestPrefix)
	requests := make(chan components.V2ProjectsListProjectsRequestBody, 10)
	cmd := pickerServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/projects.listProjects" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var request components.V2ProjectsListProjectsRequestBody
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		requests <- request
		response := components.V2ProjectsListProjectsResponseBody{}
		switch {
		case request.Search != nil && *request.Search == "needle" && request.Cursor == nil:
			response.Data = []components.Project{{ID: foundID, Name: "Needle"}}
		case request.Search == nil && request.Cursor == nil:
			response.Data = []components.Project{{ID: firstID, Name: "Billing\x1b[31m\n\u202e"}}
			response.Pagination = components.Pagination{HasMore: true, Cursor: &cursor}
		case request.Search == nil && request.Cursor != nil && *request.Cursor == cursor:
			response.Data = []components.Project{{ID: secondID, Name: "Second page"}}
		default:
			t.Errorf("unexpected query: %+v", request)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		pickerRespond(t, w, response)
	})
	script := &pickerScript{t: t, answers: []string{"browse", "next", "search", "needle", "id:" + foundID}}
	var output bytes.Buffer
	f := &form{prompt: script, out: &output}
	value, handled, err := resourcePicker(t.Context(), cmd, "projects update-project")(t.Context(), f, "project", map[string]any{})
	require.NoError(t, err)
	require.True(t, handled)
	require.Equal(t, foundID, value)
	require.Empty(t, script.answers)
	require.Len(t, requests, 3)
	first, second, searched := <-requests, <-requests, <-requests
	require.Nil(t, first.Cursor)
	require.Nil(t, first.Search)
	require.Equal(t, &cursor, second.Cursor)
	require.Nil(t, second.Search)
	require.Nil(t, searched.Cursor)
	require.Equal(t, new("needle"), searched.Search)
	require.Equal(t, new(int64(20)), first.Limit)
	require.Equal(t, "Billing [31m   ("+firstID+")", script.menus[1][0].Label)
	require.Contains(t, script.menus[1], prompt.SelectOption{Key: "next", Label: "Next page"})
	require.NotContains(t, script.menus[3], prompt.SelectOption{Key: "next", Label: "Next page"})
	require.Empty(t, output.String())
}

func TestResourcePickerManualNeedsNeitherCredentialsNorParentReads(t *testing.T) {
	for _, field := range []string{"project", "app", "environment", "keyId", "permissionId", "permission", "roleId", "role", "identityId", "identity", "namespaceId", "namespace", "overrideId", "deploymentId", "domainId", "policyId"} {
		t.Run(field, func(t *testing.T) {
			id := uid.New(uid.TestPrefix)
			script := &pickerScript{t: t, answers: []string{"manual", "", id}}
			var output bytes.Buffer
			f := &form{prompt: script, out: &output}
			value, handled, err := resourcePicker(t.Context(), nil, "")(t.Context(), f, field, map[string]any{})
			require.NoError(t, err)
			require.True(t, handled)
			require.Equal(t, id, value)
			require.Empty(t, script.answers)
			require.Len(t, script.menus, 1)
			require.Contains(t, output.String(), "Enter an ID")
		})
	}
}

func TestResourcePickerUnsupportedTargetsLeaveTextPromptToForm(t *testing.T) {
	for _, field := range []string{"apiId", "portalId", "portal", "domain", "id", "unknown"} {
		t.Run(field, func(t *testing.T) {
			value, handled, err := resourcePicker(t.Context(), nil, "")(t.Context(), nil, field, nil)
			require.NoError(t, err)
			require.False(t, handled)
			require.Empty(t, value)
		})
	}
}

func TestResourcePickerListFailureFallsBackWithoutLeakingResponse(t *testing.T) {
	secret := uid.New(uid.TestPrefix)
	id := uid.New(uid.ProjectPrefix)
	cmd := pickerServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		pickerRespond(t, w, apierrors.ForbiddenErrorResponse{
			Error_: components.BaseError{Detail: secret + "\x1b[31m", Status: http.StatusForbidden},
		})
	})
	script := &pickerScript{t: t, answers: []string{"browse", id}}
	var output bytes.Buffer
	f := &form{prompt: script, out: &output}
	value, handled, err := resourcePicker(t.Context(), cmd, "projects update-project")(t.Context(), f, "project", nil)
	require.NoError(t, err)
	require.True(t, handled)
	require.Equal(t, id, value)
	require.Empty(t, script.answers)
	require.Contains(t, output.String(), "enter the ID manually")
	require.NotContains(t, output.String(), secret)
	require.NotContains(t, output.String(), "\x1b")
}

func TestResourcePickerScopesNeverBecomeRequestFields(t *testing.T) {
	projectID, appID, environmentID := uid.New(uid.ProjectPrefix), uid.New(uid.AppPrefix), uid.New(uid.EnvironmentPrefix)
	policyID := uid.New(uid.PolicyPrefix)
	for _, provided := range []bool{false, true} {
		t.Run(fmt.Sprintf("parents_provided_%t", provided), func(t *testing.T) {
			requests := make(chan map[string]string, 10)
			cmd := pickerServer(t, func(w http.ResponseWriter, r *http.Request) {
				var request struct {
					Project     string `json:"project"`
					App         string `json:"app"`
					Environment string `json:"environment"`
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				requests <- map[string]string{"path": r.URL.Path, "project": request.Project, "app": request.App, "environment": request.Environment}
				switch r.URL.Path {
				case "/v2/projects.listProjects":
					pickerRespond(t, w, components.V2ProjectsListProjectsResponseBody{Data: []components.Project{{ID: projectID, Name: "Payments"}}})
				case "/v2/apps.listApps":
					pickerRespond(t, w, components.V2AppsListAppsResponseBody{Data: []components.App{{ID: appID, Name: "Gateway"}}})
				case "/v2/environments.listEnvironments":
					pickerRespond(t, w, components.V2EnvironmentsListEnvironmentsResponseBody{Data: []components.Environment{{ID: environmentID, Slug: "production", Kind: components.EnvironmentKindProduction}}})
				case "/v2/gateway.listPolicies":
					pickerRespond(t, w, components.V2GatewayListPoliciesResponseBody{Data: []components.PolicyResponse{{ID: policyID, Name: "Authentication"}}})
				default:
					t.Errorf("unexpected path: %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			})
			values := map[string]any{"name": "New policy name"}
			answers := []string{"browse", "browse", "id:" + projectID, "browse", "id:" + appID, "browse", "id:" + environmentID, "id:" + policyID}
			if provided {
				values["project"], values["app"], values["environment"] = projectID, appID, environmentID
				answers = []string{"browse", "id:" + policyID}
			}
			before := maps.Clone(values)
			script := &pickerScript{t: t, answers: answers}
			var output bytes.Buffer
			f := &form{prompt: script, out: &output, pick: resourcePicker(t.Context(), cmd, "gateway update-policy")}
			require.NoError(t, f.field(t.Context(), &Schema{Type: "string"}, values, "policyId", ""))
			before["policyId"] = policyID
			require.Equal(t, before, values)
			require.Empty(t, script.answers)
			if !provided {
				require.Len(t, requests, 4)
				require.Equal(t, map[string]string{"path": "/v2/projects.listProjects", "project": "", "app": "", "environment": ""}, <-requests)
				require.Equal(t, map[string]string{"path": "/v2/apps.listApps", "project": projectID, "app": "", "environment": ""}, <-requests)
				require.Equal(t, map[string]string{"path": "/v2/environments.listEnvironments", "project": projectID, "app": appID, "environment": ""}, <-requests)
			}
			require.Len(t, requests, 1)
			require.Equal(t, map[string]string{"path": "/v2/gateway.listPolicies", "project": projectID, "app": appID, "environment": environmentID}, <-requests)
		})
	}
}

func TestResourcePickerKeysScopeAndSecretRedaction(t *testing.T) {
	for _, root := range []bool{false, true} {
		t.Run(fmt.Sprintf("root_%t", root), func(t *testing.T) {
			apiID, keyID := uid.New(uid.APIPrefix), uid.New(uid.KeyPrefix)
			secret, start, end := uid.New(uid.TestPrefix), uid.New(uid.TestPrefix), uid.New(uid.TestPrefix)
			requests := make(chan map[string]any, 10)
			path, legacy := "/v2/apis.listKeys", "keys update-key"
			answers := []string{"browse", apiID, "id:" + keyID}
			if root {
				path, legacy = "/v2/rootKeys.listKeys", "root-keys update-key"
				answers = []string{"browse", "id:" + keyID}
			}
			cmd := pickerServer(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != path {
					t.Errorf("path = %s, want %s", r.URL.Path, path)
					w.WriteHeader(http.StatusNotFound)
					return
				}
				var request map[string]any
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				requests <- request
				if root {
					pickerRespond(t, w, components.V2RootKeysListKeysResponseBody{Data: []components.V2RootKeysListKeysResponseData{{KeyID: keyID, Name: new("CI"), Start: start, End: end}}})
					return
				}
				pickerRespond(t, w, components.V2ApisListKeysResponseBody{Data: []components.KeyResponseData{{KeyID: keyID, Name: new("CI"), Plaintext: &secret, Start: start}}})
			})
			script := &pickerScript{t: t, answers: answers}
			var output bytes.Buffer
			values := map[string]any{"enabled": false}
			f := &form{prompt: script, out: &output, pick: resourcePicker(t.Context(), cmd, legacy)}
			require.NoError(t, f.field(t.Context(), &Schema{Type: "string"}, values, "keyId", ""))
			require.Equal(t, map[string]any{"enabled": false, "keyId": keyID}, values)
			require.Empty(t, script.answers)
			require.Len(t, requests, 1)
			request := <-requests
			if root {
				require.NotContains(t, request, "apiId")
			} else {
				require.Equal(t, apiID, request["apiId"])
				require.Equal(t, false, request["decrypt"])
			}
			require.Equal(t, []prompt.SelectOption{{Key: "id:" + keyID, Label: "CI (" + keyID + ")"}, {Key: "manual", Label: "Enter ID manually"}}, script.menus[1])
			require.NotContains(t, fmt.Sprint(script.menus)+output.String(), secret)
			require.NotContains(t, fmt.Sprint(script.menus)+output.String(), start)
			require.NotContains(t, fmt.Sprint(script.menus)+output.String(), end)
		})
	}
}

func TestResourcePickerEmptyListAllowsManualEntryAndSearchClear(t *testing.T) {
	requests := make(chan components.V2ProjectsListProjectsRequestBody, 10)
	cmd := pickerServer(t, func(w http.ResponseWriter, r *http.Request) {
		var request components.V2ProjectsListProjectsRequestBody
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		requests <- request
		pickerRespond(t, w, components.V2ProjectsListProjectsResponseBody{})
	})
	id := uid.New(uid.ProjectPrefix)
	script := &pickerScript{t: t, answers: []string{"browse", "search", "missing", "search", "", "manual", id}}
	var output bytes.Buffer
	f := &form{prompt: script, out: &output}
	value, handled, err := resourcePicker(t.Context(), cmd, "projects update-project")(t.Context(), f, "project", nil)
	require.NoError(t, err)
	require.True(t, handled)
	require.Equal(t, id, value)
	require.Empty(t, script.answers)
	require.Contains(t, script.labels, "Project · No resources found")
	require.Len(t, requests, 3)
	require.Nil(t, (<-requests).Search)
	require.Equal(t, new("missing"), (<-requests).Search)
	require.Nil(t, (<-requests).Search)
}

func TestResourcePickerCancellationDoesNotOfferManualFallback(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	cmd := pickerServer(t, func(w http.ResponseWriter, r *http.Request) {
		cancel()
		w.WriteHeader(http.StatusForbidden)
	})
	script := &pickerScript{t: t, answers: []string{"browse"}}
	var output bytes.Buffer
	f := &form{prompt: script, out: &output}
	value, handled, err := resourcePicker(ctx, cmd, "projects update-project")(ctx, f, "project", nil)
	require.ErrorIs(t, err, context.Canceled)
	require.True(t, handled)
	require.Empty(t, value)
	require.Empty(t, output.String())
}

func TestResourcePickerResourceMappingsStopAfterSelection(t *testing.T) {
	projectID, appID, environmentID := uid.New(uid.ProjectPrefix), uid.New(uid.AppPrefix), uid.New(uid.EnvironmentPrefix)
	namespaceID := uid.New(uid.RatelimitNamespacePrefix)
	parents := map[string]any{"project": projectID, "app": appID, "environment": environmentID}
	cases := []struct {
		fields     []string
		path       string
		prefix     uid.Prefix
		name       string
		searchable bool
		scope      map[string]any
		data       func(string) any
	}{
		{[]string{"permission", "permissionId"}, "/v2/permissions.listPermissions", uid.PermissionPrefix, "Read invoices", true, nil, func(id string) any {
			return []components.Permission{{ID: id, Name: "Read invoices"}}
		}},
		{[]string{"role", "roleId"}, "/v2/permissions.listRoles", uid.RolePrefix, "Billing admin", true, nil, func(id string) any {
			return []components.Role{{ID: id, Name: "Billing admin"}}
		}},
		{[]string{"identity", "identityId"}, "/v2/identities.listIdentities", uid.IdentityPrefix, "Customer", true, nil, func(id string) any {
			return []components.Identity{{ID: id, ExternalID: "Customer"}}
		}},
		{[]string{"namespace", "namespaceId"}, "/v2/ratelimit.listNamespaces", uid.RatelimitNamespacePrefix, "API requests", true, nil, func(id string) any {
			return []components.RatelimitNamespace{{ID: id, Name: "API requests"}}
		}},
		{[]string{"overrideId"}, "/v2/ratelimit.listOverrides", uid.RatelimitOverridePrefix, "premium_*", false, map[string]any{"namespace": namespaceID}, func(id string) any {
			return []components.RatelimitOverride{{OverrideID: id, Identifier: "premium_*"}}
		}},
		{[]string{"domainId"}, "/v2/domains.listDomains", uid.DomainPrefix, "api.example.com", true, parents, func(id string) any {
			return []components.Domain{{ID: id, Domain: "api.example.com", Status: components.DomainStatusVerified}}
		}},
		{[]string{"deploymentId"}, "/v2/deployments.listDeployments", uid.DeploymentPrefix, "Gateway / production · ready", false, parents, func(id string) any {
			return []components.Deployment{{
				ID: id, App: "Gateway", Environment: "production", Status: components.DeploymentStatusReady,
				Trigger: components.DeploymentTrigger{Via: components.ViaCli},
				Runtime: components.DeploymentRuntime{ShutdownSignal: components.EnvironmentShutdownSignalSigterm, UpstreamProtocol: components.EnvironmentUpstreamProtocolHttp1},
			}}
		}},
	}
	for _, test := range cases {
		for _, field := range test.fields {
			t.Run(field, func(t *testing.T) {
				id, cursor := uid.New(test.prefix), uid.New(uid.TestPrefix)
				requests := make(chan map[string]any, 10)
				cmd := pickerServer(t, func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != test.path || r.Method != http.MethodPost {
						t.Errorf("request = %s %s, want POST %s", r.Method, r.URL.Path, test.path)
						w.WriteHeader(http.StatusNotFound)
						return
					}
					var request map[string]any
					if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
						t.Error(err)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					requests <- request
					pickerRespond(t, w, struct {
						Data       any                   `json:"data"`
						Pagination components.Pagination `json:"pagination"`
					}{Data: test.data(id), Pagination: components.Pagination{HasMore: true, Cursor: &cursor}})
				})
				script := &pickerScript{t: t, answers: []string{"browse", "id:" + id}}
				var output bytes.Buffer
				f := &form{prompt: script, out: &output}
				before := maps.Clone(test.scope)
				value, handled, err := resourcePicker(t.Context(), cmd, "")(t.Context(), f, field, test.scope)
				require.NoError(t, err)
				require.True(t, handled)
				require.Equal(t, id, value)
				require.Equal(t, before, test.scope)
				require.Empty(t, script.answers)
				require.Len(t, requests, 1)
				request := <-requests
				expected := map[string]any{"limit": float64(20)}
				maps.Copy(expected, test.scope)
				require.Equal(t, expected, request)
				require.Equal(t, prompt.SelectOption{Key: "id:" + id, Label: test.name + " (" + id + ")"}, script.menus[1][0])
				require.Contains(t, script.menus[1], prompt.SelectOption{Key: "next", Label: "Next page"})
				require.Equal(t, test.searchable, slices.ContainsFunc(script.menus[1], func(option prompt.SelectOption) bool { return option.Key == "search" }))
			})
		}
	}
}

type pickerScript struct {
	t       *testing.T
	answers []string
	labels  []string
	menus   [][]prompt.SelectOption
}

func (s *pickerScript) String(question string, _ ...string) (string, error) {
	s.t.Helper()
	s.labels = append(s.labels, question)
	if len(s.answers) == 0 {
		return "", fmt.Errorf("unexpected prompt: %s", question)
	}
	answer := s.answers[0]
	s.answers = s.answers[1:]
	return answer, nil
}

func (s *pickerScript) Secret(question string) (string, error) {
	s.t.Helper()
	return "", fmt.Errorf("unexpected secret prompt: %s", question)
}

func (s *pickerScript) SelectOrdered(question string, options []prompt.SelectOption, _ ...string) (string, error) {
	s.t.Helper()
	s.menus = append(s.menus, slices.Clone(options))
	answer, err := s.String(question)
	if err != nil {
		return "", err
	}
	if !slices.ContainsFunc(options, func(option prompt.SelectOption) bool { return option.Key == answer }) {
		return "", fmt.Errorf("answer %q unavailable in %s: %v", answer, question, options)
	}
	return answer, nil
}

func pickerServer(t *testing.T, handler http.HandlerFunc) *cli.Command {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	key := uid.New(uid.TestPrefix)
	t.Setenv("UNKEY_ROOT_KEY", key)
	t.Setenv("UNKEY_API_BASE_URL", server.URL)
	cmd := &cli.Command{
		Name:   "picker",
		Flags:  []cli.Flag{cli.String("root-key", ""), cli.String("api-url", "")},
		Action: func(context.Context, *cli.Command) error { return nil },
	}
	require.NoError(t, cmd.Run(t.Context(), []string{"picker", "--root-key", key, "--api-url", server.URL}))
	require.True(t, cmd.FlagIsSet("root-key"))
	require.True(t, cmd.FlagIsSet("api-url"))
	require.Equal(t, server.URL, cmd.String("api-url"))
	return cmd
}

func pickerRespond(t *testing.T, w http.ResponseWriter, response any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		t.Error(err)
	}
}
