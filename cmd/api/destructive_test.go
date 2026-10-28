package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v3/models/apierrors"
	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestDestructiveCommandsRequireExplicitConfirmation(t *testing.T) {
	t.Setenv("UNKEY_OUTPUT", "")
	keyID := uid.New(uid.KeyPrefix)
	apiID := uid.New(uid.APIPrefix)
	for _, test := range []struct {
		legacy   string
		endpoint string
		flags    []string
		body     any
		targets  []string
	}{
		{"projects delete-project", "projects.deleteProject", []string{"--project=payments"}, components.V2ProjectsDeleteProjectRequestBody{Project: "payments"}, []string{"payments", "all", "environments"}},
		{"apps delete-app", "apps.deleteApp", []string{"--project=payments", "--app=worker"}, components.V2AppsDeleteAppRequestBody{Project: "payments", App: "worker"}, []string{"payments", "worker", "all", "environments"}},
		{"apis delete-api", "apis.deleteApi", []string{"--api-id=" + apiID}, components.V2ApisDeleteAPIRequestBody{APIID: apiID}, []string{apiID, "keys"}},
		{"domains delete-domain", "domains.deleteDomain", []string{"--domain=api.example.com"}, components.V2DomainsDeleteDomainRequestBody{Domain: "api.example.com"}, []string{"api.example.com"}},
		{"identities delete-identity", "identities.deleteIdentity", []string{"--identity=customer"}, components.V2IdentitiesDeleteIdentityRequestBody{Identity: "customer"}, []string{"customer"}},
		{"keys delete-key", "keys.deleteKey", []string{"--key-id=" + keyID, "--permanent"}, components.V2KeysDeleteKeyRequestBody{KeyID: keyID, Permanent: new(true)}, []string{keyID, "Permanently"}},
		{"permissions delete-permission", "permissions.deletePermission", []string{"--permission=documents.read"}, components.V2PermissionsDeletePermissionRequestBody{Permission: "documents.read"}, []string{"documents.read"}},
		{"permissions delete-role", "permissions.deleteRole", []string{"--role=admin"}, components.V2PermissionsDeleteRoleRequestBody{Role: "admin"}, []string{"admin"}},
		{"portal delete-portal", "portal.deletePortal", []string{"--portal=customer-portal"}, components.V2PortalDeletePortalRequestBody{Portal: "customer-portal"}, []string{"customer-portal"}},
		{"ratelimit delete-override", "ratelimit.deleteOverride", []string{"--namespace=requests", "--identifier=premium_*"}, components.V2RatelimitDeleteOverrideRequestBody{Namespace: "requests", Identifier: "premium_*"}, []string{"requests", "premium_*"}},
		{"ratelimit delete-namespace", "ratelimit.deleteNamespace", []string{"--namespace=requests"}, components.V2RatelimitDeleteNamespaceRequestBody{Namespace: "requests"}, []string{"requests"}},
		{"root-keys delete-key", "rootKeys.deleteKey", []string{"--key-id=" + keyID}, components.V2RootKeysDeleteKeyRequestBody{KeyID: keyID}, []string{keyID}},
		{"environments remove-environment-variables", "environments.removeEnvironmentVariables", []string{"--project=payments", "--app=worker", "--environment=production", "--variables=OLD_TOKEN,LEGACY_URL"}, components.V2EnvironmentsRemoveEnvironmentVariablesRequestBody{Project: "payments", App: "worker", Environment: "production", Variables: []string{"OLD_TOKEN", "LEGACY_URL"}}, []string{"payments", "worker", "production", "OLD_TOKEN", "LEGACY_URL"}},
		{"keys remove-permissions", "keys.removePermissions", []string{"--key-id=" + keyID, "--permissions=documents.read,documents.write"}, components.V2KeysRemovePermissionsRequestBody{KeyID: keyID, Permissions: []string{"documents.read", "documents.write"}}, []string{keyID, "documents.read", "documents.write"}},
		{"keys remove-roles", "keys.removeRoles", []string{"--key-id=" + keyID, "--roles=admin,reader"}, components.V2KeysRemoveRolesRequestBody{KeyID: keyID, Roles: []string{"admin", "reader"}}, []string{keyID, "admin", "reader"}},
	} {
		t.Run(test.legacy, func(t *testing.T) {
			var requests [][]byte
			requestID := uid.New(uid.RequestPrefix)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v2/"+test.endpoint {
					t.Errorf("unexpected endpoint: %s", r.URL.Path)
				}
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
				}
				requests = append(requests, body)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				if err := json.NewEncoder(w).Encode(apierrors.ForbiddenErrorResponse{
					Meta: components.Meta{RequestID: requestID}, Error_: components.BaseError{Detail: "Missing permission."},
				}); err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(server.Close)
			body, err := json.Marshal(test.body)
			require.NoError(t, err)
			for _, path := range []string{resourcePaths[test.legacy], "api " + test.legacy} {
				for _, flags := range [][]string{test.flags, {"--body=" + string(body)}} {
					args := append(strings.Fields(path), "--root-key="+uid.New(uid.KeyPrefix), "--api-url="+server.URL)
					args = append(args, flags...)
					count := len(requests)
					output, diagnostic, err := runCLI(t, args...)
					require.ErrorContains(t, err, "--yes")
					for _, target := range test.targets {
						require.ErrorContains(t, err, target)
					}
					require.Empty(t, output)
					require.Empty(t, diagnostic)
					require.Len(t, requests, count, "no mutation before confirmation")

					output, diagnostic, err = runCLI(t, append(args, "--yes", "--json")...)
					require.Error(t, err)
					require.Contains(t, []string{
						"Permission denied: Missing permission.\nRequest ID: " + requestID,
						"API request failed (HTTP 403)",
					}, err.Error())
					require.Empty(t, output)
					require.Empty(t, diagnostic)
					require.Len(t, requests, count+1)
					require.JSONEq(t, string(body), string(requests[count]))
				}
			}
		})
	}
}

func TestConfirmationDoesNotDisableDeleteProtection(t *testing.T) {
	t.Setenv("UNKEY_OUTPUT", "")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/projects.deleteProject" {
			t.Errorf("unexpected endpoint: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusPreconditionFailed)
		if err := json.NewEncoder(w).Encode(apierrors.PreconditionFailedErrorResponse{
			Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)}, Error_: components.BaseError{Detail: "Delete protection is enabled."},
		}); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	output, _, err := runCLI(t, "projects", "delete", "--project=payments", "--yes", "--root-key="+uid.New(uid.KeyPrefix), "--api-url="+server.URL)
	require.ErrorContains(t, err, "Delete protection is enabled.")
	require.Empty(t, output)
}
