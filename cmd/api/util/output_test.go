package util

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestOutputWhoamiTreePreservesKeyDetails(t *testing.T) {
	keyID := uid.New(uid.KeyPrefix)
	identityID := uid.New(uid.IdentityPrefix)
	rateID := uid.New(uid.RatelimitPrefix)
	response := &components.V2KeysWhoamiResponseBody{
		Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)},
		Data: components.KeyResponseData{
			KeyID: keyID, Name: new("CI\n\x1b[2J"), Start: "demo_", Enabled: false,
			CreatedAt: 1700000000000, LastUsedAt: new(int64(1700003600000)),
			UpdatedAt: new(int64(1700007200000)), Expires: new(int64(1700010800000)),
			Roles: []string{"reader"}, Permissions: []string{"documents.read", "documents.list"},
			Credits: &components.KeyCreditsData{
				Remaining: new(int64(0)),
				Refill:    &components.KeyCreditsRefill{Amount: 100, Interval: components.KeyCreditsRefillIntervalMonthly, RefillDay: new(int64(3))},
			},
			Identity:   &components.Identity{ID: identityID, ExternalID: "demo-user", Meta: map[string]any{"createdAt": int64(123)}},
			Ratelimits: []components.RatelimitResponse{{ID: rateID, Name: "requests", Limit: 50, Duration: 60000, AutoApply: false}},
			Meta: map[string]any{
				"quota":  int64(9007199254740993),
				"nested": []any{map[string]any{"createdAt": int64(456), "enabled": false, "credits": map[string]any{"remaining": nil}}},
			},
		},
	}
	output := captureOutput(t, response)
	require.True(t, strings.HasPrefix(output, `CI\n\x1b[2J`+"\nDisabled · "+keyID+"\nPrefix: demo_\n"), output)
	for _, want := range []string{
		"├─ Identity", identityID, "External ID: demo-user", "createdAt: 123",
		"Permissions", "documents.read", "documents.list", "Roles", "reader",
		"Remaining: 0", "Amount: 100", "Interval: monthly", "Refill day: 3",
		"requests: 50 / 1m0s", rateID, "Auto apply: false",
		"quota: 9007199254740993", "createdAt: 456", "enabled: false", "remaining: -",
		"└─ Dates", "Created at: 2023-11-14T22:13:20Z", "Last used at: 2023-11-14T23:13:20Z",
		"Updated at: 2023-11-15T00:13:20Z", "Expires: 2023-11-15T01:13:20Z",
	} {
		require.Contains(t, output, want)
	}
	require.NotContains(t, output, "\x1b")
	require.NotContains(t, output, response.Meta.RequestID)
	for _, flags := range [][]string{{"--json"}, {"--output=json"}} {
		var got bytes.Buffer
		require.NoError(t, json.Compact(&got, []byte(captureOutput(t, response, flags...))))
		wantJSON, err := json.Marshal(response)
		require.NoError(t, err)
		require.Equal(t, string(wantJSON), got.String())
	}
}

func TestOutputWhoamiTreeShowsUnlimitedCredits(t *testing.T) {
	response := &components.V2KeysWhoamiResponseBody{Data: components.KeyResponseData{
		KeyID: uid.New(uid.KeyPrefix), Enabled: true, Credits: &components.KeyCreditsData{},
	}}
	output := captureOutput(t, response)
	require.True(t, strings.HasPrefix(output, "API key\nEnabled · "), output)
	require.Contains(t, output, "Remaining: unlimited")
	require.NotContains(t, output, "Permissions")
	require.NotContains(t, output, "Identity")
}

func TestOutputDetailsPreserveValuesAndEscapeTerminalControls(t *testing.T) {
	keyID := uid.New(uid.KeyPrefix)
	requestID := uid.New(uid.RequestPrefix)
	plaintext := strings.Repeat("a", 120)
	response := &components.V2KeysGetKeyResponseBody{
		Meta: components.Meta{RequestID: requestID},
		Data: components.KeyResponseData{
			KeyID: keyID, Name: new("Payments\n\x1b[2J"), Enabled: false,
			CreatedAt: 1700000000000, Plaintext: &plaintext,
			Credits: &components.KeyCreditsData{Remaining: new(int64(0))},
			Meta:    map[string]any{"quota": int64(9007199254740993), "createdAt": int64(123)},
			Roles:   []string{"billing", "reader"},
		},
	}
	output := captureOutput(t, response)
	for _, want := range []string{keyID, plaintext, "false", "Remaining:  0", "createdAt:  123", "9007199254740993", "2023-11-14T22:13:20Z", "billing", "reader", `Payments\n\x1b[2J`} {
		require.Contains(t, output, want)
	}
	require.NotContains(t, output, requestID)
	require.NotContains(t, output, "\x1b")
	require.Equal(t, 1, strings.Count(output, plaintext))
}

func TestOutputKeyListShowsReturnedPlaintext(t *testing.T) {
	plaintext := strings.Repeat("secret", 30)
	for _, test := range []struct {
		name      string
		plaintext *string
	}{
		{"without decryption", nil},
		{"with decryption", &plaintext},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := &components.V2ApisListKeysResponseBody{
				Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)},
				Data: []components.KeyResponseData{
					{KeyID: uid.New(uid.KeyPrefix), Name: new("Unrecoverable"), Enabled: false},
					{KeyID: uid.New(uid.KeyPrefix), Name: new("Recoverable"), Enabled: true, Plaintext: test.plaintext},
				},
			}
			output := captureOutput(t, response)
			if test.plaintext == nil {
				require.NotContains(t, output, "PLAINTEXT")
				require.NotContains(t, output, plaintext)
			} else {
				require.Contains(t, output, "PLAINTEXT")
				require.Contains(t, output, plaintext)
			}
		})
	}
}

func TestOutputAnalyticsPreservesEveryColumnAndLargeNumbers(t *testing.T) {
	response := &components.V2AnalyticsGetGatewayRequestsResponseBody{
		Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)},
		Data: []map[string]any{
			{"path": "/v1/orders", "count": int64(9007199254740993)},
			{"path": "/v1/refunds", "count": int64(0), "allowed": false},
		},
	}
	output := captureOutput(t, response)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	require.Len(t, lines, 3)
	require.Equal(t, []string{"ALLOWED", "COUNT", "PATH"}, strings.Fields(lines[0]))
	require.Equal(t, []string{"-", "9007199254740993", "/v1/orders"}, strings.Fields(lines[1]))
	require.Equal(t, []string{"false", "0", "/v1/refunds"}, strings.Fields(lines[2]))
}

func TestOutputSummariesKeepActionableFields(t *testing.T) {
	keyID := uid.New(uid.KeyPrefix)
	overrideID := uid.New(uid.RatelimitOverridePrefix)
	deploymentID := uid.New(uid.DeploymentPrefix)
	for _, test := range []struct {
		name     string
		response any
		header   string
		values   []string
	}{
		{
			name: "deployments across projects",
			response: &components.V2DeploymentsListDeploymentsResponseBody{Data: []components.Deployment{
				{ID: deploymentID, Project: "payments", App: "api", Environment: "production", Status: "ready", IsCurrent: true, CreatedAt: 1700003600000},
			}},
			header: "PROJECT APP ENVIRONMENT STATUS CURRENT CREATED ID",
			values: []string{"payments", "api", "production", "ready", "Yes", "2023-11-14T23:13:20Z", deploymentID},
		},
		{
			name: "keys",
			response: &components.V2ApisListKeysResponseBody{Data: []components.KeyResponseData{
				{KeyID: keyID, Name: new("Payments"), Start: "unkey_pay", Enabled: false, LastUsedAt: new(int64(1700003600000))},
			}},
			header: "NAME START STATUS LAST USED KEY ID",
			values: []string{"Payments", "unkey_pay", "Disabled", "2023-11-14T23:13:20Z", keyID},
		},
		{
			name: "permissions",
			response: &components.V2PermissionsListPermissionsResponseBody{Data: []components.Permission{
				{ID: uid.New(uid.PermissionPrefix), Name: "Read payments", Slug: "payments.read", Description: new("Read payment details")},
			}},
			header: "NAME SLUG DESCRIPTION",
			values: []string{"Read payments", "payments.read", "Read payment details"},
		},
		{
			name: "roles",
			response: &components.V2PermissionsListRolesResponseBody{Data: []components.Role{
				{ID: uid.New(uid.RolePrefix), Name: "Billing administrator", Description: new("Manage billing")},
			}},
			header: "NAME DESCRIPTION",
			values: []string{"Billing administrator", "Manage billing"},
		},
		{
			name: "rate limit overrides",
			response: &components.V2RatelimitListOverridesResponseBody{Data: []components.RatelimitOverride{
				{OverrideID: overrideID, Identifier: "customer@example.com", Limit: 0, Duration: 1500},
			}},
			header: "IDENTIFIER LIMIT DURATION OVERRIDE ID",
			values: []string{"customer@example.com", "0", "1.5s", overrideID},
		},
		{
			name: "root keys",
			response: &components.V2RootKeysListKeysResponseBody{Data: []components.V2RootKeysListKeysResponseData{
				{KeyID: keyID, Name: new("Automation"), Start: "unkey_auto", Enabled: false, Expires: new(int64(1700003600000)), LastUsedAt: 0},
			}},
			header: "NAME START STATUS EXPIRES LAST USED KEY ID",
			values: []string{"Automation", "unkey_auto", "Disabled", "2023-11-14T23:13:20Z", keyID},
		},
		{
			name: "rate limit namespaces",
			response: &components.V2RatelimitListNamespacesResponseBody{Data: []components.RatelimitNamespace{
				{ID: uid.New(uid.RatelimitNamespacePrefix), Name: "email.outbound", CreatedAt: 1700003600000},
			}},
			header: "NAME CREATED ID",
			values: []string{"email.outbound", "2023-11-14T23:13:20Z"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			output := captureOutput(t, test.response)
			header, row, found := strings.Cut(output, "\n")
			require.True(t, found)
			require.Equal(t, test.header, strings.Join(strings.Fields(header), " "))
			for _, value := range test.values {
				require.Contains(t, row, value)
			}
		})
	}
}

func TestOutputEmptyResponses(t *testing.T) {
	tests := []struct {
		name     string
		response any
		want     string
	}{
		{"empty projects", &components.V2ProjectsListProjectsResponseBody{Data: []components.Project{}}, "No projects found.\n"},
		{"nil projects", &components.V2ProjectsListProjectsResponseBody{}, "No projects found.\n"},
		{"empty role list", &components.V2PermissionsListRolesResponseBody{Data: []components.Role{}}, "No roles found.\n"},
		{"empty permission list", &components.V2PermissionsListPermissionsResponseBody{}, "No permissions found.\n"},
		{"roles cleared", &components.V2KeysSetRolesResponseBody{Data: []components.Role{}}, "✓ Updated roles\n  No roles assigned.\n"},
		{"nil roles", &components.V2KeysSetRolesResponseBody{}, "✓ Updated roles\n  No roles assigned.\n"},
		{"roles removed", &components.V2KeysRemoveRolesResponseBody{}, "✓ Updated roles\n  No roles assigned.\n"},
		{"roles added", &components.V2KeysAddRolesResponseBody{}, "✓ Updated roles\n  No roles assigned.\n"},
		{"permissions cleared", &components.V2KeysSetPermissionsResponseBody{Data: []components.Permission{}}, "✓ Updated permissions\n  No permissions assigned.\n"},
		{"permissions removed", &components.V2KeysRemovePermissionsResponseBody{}, "✓ Updated permissions\n  No permissions assigned.\n"},
		{"permissions added", &components.V2KeysAddPermissionsResponseBody{}, "✓ Updated permissions\n  No permissions assigned.\n"},
		{"role permissions cleared", &components.V2PermissionsSetRolePermissionsResponseBody{}, "✓ Updated permissions\n  No permissions assigned.\n"},
		{"no analytics results", &components.V2AnalyticsGetRatelimitsResponseBody{}, "No results found.\n"},
		{"queued deletion", &components.V2ProjectsDeleteProjectResponseBody{}, "✓ Deletion requested\n"},
		{"queued stop", &components.V2DeploymentsStopDeploymentResponseBody{}, "✓ Stop requested\n"},
		{"completed mutation", &components.V2KeysDeleteKeyResponseBody{}, "✓ Deleted key\n"},
		{"absent data", &components.V2IdentitiesDeleteIdentityResponseBody{}, "✓ Deleted identity\n"},
		{"root keys", &components.V2RootKeysListKeysResponseBody{}, "No root keys found.\n"},
		{"build logs", &components.V2DeploymentsListBuildLogsResponseBody{}, "No build logs found.\n"},
		{"API requests", &components.V2LogsListAPIRequestsResponseBody{}, "No API requests found.\n"},
		{"request metrics", &components.V2LogsGetAPIRequestMetricsResponseBody{}, "No API request metrics found.\n"},
		{"rate limit namespaces", &components.V2RatelimitListNamespacesResponseBody{}, "No rate limit namespaces found.\n"},
		{"workspace flags", &components.V2WorkspaceGetFlagsResponseBody{}, "No workspace flags found.\n"},
		{"root key deletion", &components.V2RootKeysDeleteKeyResponseBody{}, "✓ Deleted root key\n"},
		{"root key update", &components.V2RootKeysUpdateKeyResponseBody{}, "✓ Updated root key\n"},
		{"namespace deletion", &components.V2RatelimitDeleteNamespaceResponseBody{}, "✓ Deleted rate limit namespace\n"},
		{"queued cancellation", &components.V2DeploymentsCancelDeploymentResponseBody{}, "✓ Cancellation requested\n"},
		{"queued authorization", &components.V2DeploymentsAuthorizeDeploymentResponseBody{}, "✓ Authorization requested\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, captureOutput(t, test.response))
		})
	}
}

func TestOutputMutationStatusPreservesReturnedData(t *testing.T) {
	keyID := uid.New(uid.KeyPrefix)
	projectID := uid.New(uid.ProjectPrefix)
	deploymentID := uid.New(uid.DeploymentPrefix)
	plaintext := strings.Repeat("returned-secret", 10)
	for _, test := range []struct {
		name     string
		response any
		status   string
		values   []string
	}{
		{
			name:     "created key",
			response: &components.V2KeysCreateKeyResponseBody{Data: components.V2KeysCreateKeyResponseData{KeyID: keyID, Key: plaintext}},
			status:   "✓ Created key",
			values:   []string{keyID, plaintext},
		},
		{
			name:     "updated project",
			response: &components.V2ProjectsUpdateProjectResponseBody{Data: components.Project{ID: projectID, Name: "Payments", Slug: "payments"}},
			status:   "✓ Updated project",
			values:   []string{projectID, "Payments", "payments"},
		},
		{
			name:     "queued deployment",
			response: &components.V2DeploymentsCreateDeploymentResponseBody{Data: components.V2DeploymentsCreateDeploymentResponseData{DeploymentID: deploymentID}},
			status:   "✓ Deployment requested",
			values:   []string{deploymentID},
		},
		{
			name:     "queued v3 deployment",
			response: &components.V3DeploymentsCreateDeploymentResponseBody{Data: components.V3DeploymentsCreateDeploymentResponseData{DeploymentID: deploymentID}},
			status:   "✓ Deployment requested",
			values:   []string{deploymentID},
		},
		{
			name:     "assigned roles",
			response: &components.V2KeysSetRolesResponseBody{Data: []components.Role{{ID: uid.New(uid.RolePrefix), Name: "Billing reader"}}},
			status:   "✓ Updated roles",
			values:   []string{"NAME", "Billing reader"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			output := captureOutput(t, test.response)
			require.True(t, strings.HasPrefix(output, test.status+"\n"), output)
			for _, value := range test.values {
				require.Contains(t, output, value)
			}
			require.Equal(t, 1, strings.Count(output, test.status))
		})
	}
}

func TestOutputCreatedResourceNamesRequestedIdentity(t *testing.T) {
	projectID := uid.New(uid.ProjectPrefix)
	response := &components.V2ProjectsCreateProjectResponseBody{Data: components.V2ProjectsCreateProjectResponseData{ID: projectID}}
	for _, test := range []struct {
		name string
		args []string
		want string
	}{
		{"slug flag", []string{"--name=Demo Service", "--slug=demo-service"}, "✓ Created project demo-service\n  ID:  " + projectID + "\n"},
		{"slug body", []string{`--body={"name":"Demo Service","slug":"demo-service"}`}, "✓ Created project demo-service\n  ID:  " + projectID + "\n"},
		{"name only", []string{`--body={"name":"Demo Service"}`}, "✓ Created project Demo Service\n  ID:  " + projectID + "\n"},
		{"control characters", []string{"--slug=demo\x1b[2J"}, "✓ Created project demo\\x1b[2J\n  ID:  " + projectID + "\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, captureOutput(t, response, test.args...))
			var got components.V2ProjectsCreateProjectResponseBody
			require.NoError(t, json.Unmarshal([]byte(captureOutput(t, response, append(test.args, "--json")...)), &got))
			require.Equal(t, *response, got)
		})
	}
}

func TestOutputKeyVerificationDoesNotClaimMutationSuccess(t *testing.T) {
	for _, code := range []components.Code{components.CodeValid, components.CodeNotFound} {
		t.Run(string(code), func(t *testing.T) {
			response := &components.V2KeysVerifyKeyResponseBody{
				Data: components.V2KeysVerifyKeyResponseData{Valid: code == components.CodeValid, Code: code},
			}
			output := captureOutput(t, response)
			require.True(t, strings.HasPrefix(output, "Code:"), output)
			require.Contains(t, output, string(code))
			require.Contains(t, output, "Valid:")
			require.NotContains(t, output, "Done.")
			require.NotContains(t, output, "requested.")
		})
	}
}

func TestOutputNoDataMutationsIdentifyOnlySafeTargets(t *testing.T) {
	keyID := uid.New(uid.KeyPrefix)
	deploymentID := uid.New(uid.DeploymentPrefix)
	secret := strings.Repeat("request-only-secret", 10)
	variables := []components.EnvironmentVariableInput{{Key: "TOKEN", Value: secret}}
	variablesJSON, err := json.Marshal(variables)
	require.NoError(t, err)
	for _, test := range []struct {
		name     string
		response any
		request  any
		flags    []string
		want     string
	}{
		{
			name:     "key deletion",
			response: &components.V2KeysDeleteKeyResponseBody{},
			request:  components.V2KeysDeleteKeyRequestBody{KeyID: keyID},
			flags:    []string{"--key-id=" + keyID},
			want:     "✓ Deleted key\n  Key ID: " + keyID + "\n",
		},
		{
			name:     "queued stop",
			response: &components.V2DeploymentsStopDeploymentResponseBody{},
			request:  components.V2DeploymentsStopDeploymentRequestBody{DeploymentID: deploymentID},
			flags:    []string{"--deployment-id=" + deploymentID},
			want:     "✓ Stop requested\n  Deployment ID: " + deploymentID + "\n",
		},
		{
			name:     "override deletion",
			response: &components.V2RatelimitDeleteOverrideResponseBody{},
			request:  components.V2RatelimitDeleteOverrideRequestBody{Namespace: "payments", Identifier: "premium_*"},
			flags:    []string{"--namespace=payments", "--identifier=premium_*"},
			want:     "✓ Deleted rate limit override\n  Namespace: payments\n  Identifier: premium_*\n",
		},
		{
			name:     "environment values stay private",
			response: &components.V2EnvironmentsSetEnvironmentVariablesResponseBody{},
			request:  components.V2EnvironmentsSetEnvironmentVariablesRequestBody{Project: "payments", App: "api", Environment: "production", Variables: variables},
			flags:    []string{"--project=payments", "--app=api", "--environment=production", "--variables=" + string(variablesJSON)},
			want:     "✓ Saved variables\n  Project: payments\n  App: api\n  Environment: production\n",
		},
		{
			name:     "target escapes terminal controls",
			response: &components.V2ProjectsDeleteProjectResponseBody{},
			request:  components.V2ProjectsDeleteProjectRequestBody{Project: "payments\n\x1b[2J"},
			flags:    []string{"--project=payments\n\x1b[2J"},
			want:     "✓ Deletion requested\n  Project: payments\\n\\x1b[2J\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			body, err := json.Marshal(test.request)
			require.NoError(t, err)
			for _, args := range [][]string{test.flags, {"--body=" + string(body)}} {
				args = append(args, "--root-key="+secret)
				require.Equal(t, test.want, captureOutput(t, test.response, args...))
				output := captureOutput(t, test.response, append(args, "--json")...)
				encoded, err := json.MarshalIndent(test.response, "", "  ")
				require.NoError(t, err)
				require.Equal(t, string(encoded)+"\n", output)
			}
		})
	}
}

func TestOutputJSONPreservesTheEnvelope(t *testing.T) {
	response := &components.V2KeysCreateKeyResponseBody{
		Meta: components.Meta{RequestID: uid.New(uid.RequestPrefix)},
		Data: components.V2KeysCreateKeyResponseData{KeyID: uid.New(uid.KeyPrefix), Key: strings.Repeat("a", 120)},
	}
	for _, test := range []struct {
		name string
		env  string
		args []string
	}{
		{"flag", "", []string{"--output=json"}},
		{"shorthand", "", []string{"--json"}},
		{"environment", "json", nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("UNKEY_OUTPUT", test.env)
			output := captureOutput(t, response, test.args...)
			var got components.V2KeysCreateKeyResponseBody
			require.NoError(t, json.Unmarshal([]byte(output), &got))
			require.Equal(t, *response, got)
			encoded, err := json.MarshalIndent(response, "", "  ")
			require.NoError(t, err)
			require.Equal(t, string(encoded)+"\n", output)
		})
	}
}

func TestOutputLogsPreservesSelectorsAndEscapesMessages(t *testing.T) {
	requestID := uid.New(uid.RequestPrefix)
	for _, test := range []struct {
		name     string
		response any
		values   []string
	}{
		{"build logs", &components.V2DeploymentsListBuildLogsResponseBody{Data: []components.BuildLogEntry{{Time: 1700000000000, StepID: "build-step", Step: "RUN make", Output: components.BuildLogOutputStderr, Message: "first\nsecond\x1b[2J"}}}, []string{"2023-11-14T22:13:20Z", "build-step", "RUN make", "stderr", `first\nsecond\x1b[2J`}},
		{"API requests", &components.V2LogsListAPIRequestsResponseBody{Data: []components.APIRequestLog{{RequestID: requestID, Time: 1700000000000, Method: "POST", Path: "/v2/keys.verifyKey", Status: 429, LatencyMs: 17}}}, []string{"2023-11-14T22:13:20Z", "POST", "/v2/keys.verifyKey", "429", "17", requestID}},
		{"metrics", &components.V2LogsGetAPIRequestMetricsResponseBody{Data: []components.APIRequestMetricsPoint{{Time: 1700000000000, Success: 53, ClientError: 7, ServerError: 2, Total: 63}}}, []string{"2023-11-14T22:13:20Z", "53", "7", "2", "63"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			output := captureOutput(t, test.response)
			for _, value := range test.values {
				require.Contains(t, output, value)
			}
			require.NotContains(t, output, "\x1b")
		})
	}
}

func TestOutputBodyPaginationEscapesControlsWithoutChangingCursor(t *testing.T) {
	cursor := "left\u202Eright\u200F分析\x7f\U000e0001🚀\\\"\n\x1b[2J"
	response := &components.V2PermissionsListRolesResponseBody{
		Meta:       components.Meta{RequestID: uid.New(uid.RequestPrefix)},
		Data:       []components.Role{},
		Pagination: components.Pagination{Cursor: &cursor, HasMore: true},
	}
	body, err := json.Marshal(components.V2PermissionsListRolesRequestBody{})
	require.NoError(t, err)
	output := captureOutput(t, response, "--body="+string(body))
	require.Contains(t, output, "Keep the same filters in --body")
	for _, control := range []string{"\u202E", "\u200F", "\x7f", "\U000e0001", "\x1b"} {
		require.NotContains(t, output, control)
	}
	_, encoded, found := strings.Cut(output, "set cursor to ")
	require.True(t, found)
	var decoded string
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(encoded)), &decoded))
	require.Equal(t, cursor, decoded)

	output = captureOutput(t, response, "--json")
	var got components.V2PermissionsListRolesResponseBody
	require.NoError(t, json.Unmarshal([]byte(output), &got))
	require.Equal(t, *response, got)
}

func TestOutputPaginationKeepsFilters(t *testing.T) {
	cursor := "next'page"
	response := &components.V2PermissionsListRolesResponseBody{
		Pagination: components.Pagination{Cursor: &cursor, HasMore: true},
	}
	require.Contains(t, captureOutput(t, response), "Repeat this command with the same filters and --cursor='next'\"'\"'page'")
}

func TestOutputDebugDiagnosticsStayOnStderr(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "stderr")
	require.NoError(t, err)
	original := os.Stderr
	os.Stderr = file
	t.Cleanup(func() {
		os.Stderr = original
		require.NoError(t, file.Close())
	})
	requestID := uid.New(uid.RequestPrefix)
	response := &components.V2KeysDeleteKeyResponseBody{Meta: components.Meta{RequestID: requestID}}
	require.Equal(t, "✓ Deleted key\n", captureOutput(t, response, "--debug"))
	output := captureOutput(t, response, "--debug", "--json")
	var got components.V2KeysDeleteKeyResponseBody
	require.NoError(t, json.Unmarshal([]byte(output), &got))
	require.Equal(t, *response, got)
	os.Stderr = original
	diagnostics, err := os.ReadFile(file.Name())
	require.NoError(t, err)
	require.Equal(t, strings.Repeat("Request ID: "+requestID+"\n", 2), string(diagnostics))
}

func TestOutputEnvironmentSecretsRequireRevealAndJSONPreservesValues(t *testing.T) {
	secret := uid.New(uid.KeyPrefix)
	response := &components.V2EnvironmentsListEnvironmentVariablesResponseBody{Data: []components.EnvironmentVariable{
		{Key: "TOKEN", Kind: components.EnvironmentVariableKindRecoverable, Value: &secret},
		{Key: "EMPTY", Kind: components.EnvironmentVariableKindRecoverable, Value: new("")},
		{Key: "OMITTED_EMPTY", Kind: components.EnvironmentVariableKindRecoverable},
		{Key: "WRITE_ONLY", Kind: components.EnvironmentVariableKindWriteonly},
	}}
	output := captureOutput(t, response)
	require.NotContains(t, output, secret)
	require.Contains(t, output, "[hidden]")
	require.Contains(t, output, "not retrievable")
	output = captureOutput(t, response, "--reveal")
	require.Contains(t, output, secret)
	require.Equal(t, 2, strings.Count(output, "(empty)"))
	require.Contains(t, output, "not retrievable")
	for _, flags := range [][]string{{"--json"}, {"--reveal", "--output=json"}} {
		var got components.V2EnvironmentsListEnvironmentVariablesResponseBody
		require.NoError(t, json.Unmarshal([]byte(captureOutput(t, response, flags...)), &got))
		require.Equal(t, *response, got)
	}
}

func captureOutput(t *testing.T, response any, args ...string) string {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "stdout")
	require.NoError(t, err)
	original := os.Stdout
	os.Stdout = file
	t.Cleanup(func() {
		os.Stdout = original
		require.NoError(t, file.Close())
	})
	cmd := &cli.Command{
		Name: "test", Flags: []cli.Flag{
			OutputFlag(), cli.Bool("json", "JSON output"), cli.Bool("debug", "Debug diagnostics"),
			cli.Bool("reveal", "Show recoverable environment variable values"),
			cli.String("body", "JSON request body"), cli.String("root-key", "Root key"),
			cli.String("key-id", "Key ID"), cli.String("deployment-id", "Deployment ID"),
			cli.String("project", "Project"), cli.String("app", "App"), cli.String("environment", "Environment"),
			cli.String("namespace", "Namespace"), cli.String("identifier", "Identifier"),
			cli.String("variables", "Environment variables"),
			cli.String("name", "Name"), cli.String("slug", "Slug"),
		},
		Action: func(_ context.Context, cmd *cli.Command) error { return Output(cmd, response) },
	}
	require.NoError(t, cmd.Run(t.Context(), append([]string{"test"}, args...)))
	os.Stdout = original
	output, err := os.ReadFile(file.Name())
	require.NoError(t, err)
	return string(output)
}
