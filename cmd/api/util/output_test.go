package util

import (
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
		{"empty roles", &components.V2KeysSetRolesResponseBody{Data: []components.Role{}}, "No roles found.\n"},
		{"no analytics results", &components.V2AnalyticsGetRatelimitsResponseBody{}, "No results found.\n"},
		{"queued deletion", &components.V2ProjectsDeleteProjectResponseBody{}, "Deletion requested.\n"},
		{"queued stop", &components.V2DeploymentsStopDeploymentResponseBody{}, "Stop requested.\n"},
		{"completed mutation", &components.V2KeysDeleteKeyResponseBody{}, "Key deleted.\n"},
		{"absent data", &components.V2IdentitiesDeleteIdentityResponseBody{}, "Identity deleted.\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, captureOutput(t, test.response))
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
		Name: "test", Flags: []cli.Flag{OutputFlag(), cli.Bool("json", "JSON output"), cli.String("body", "JSON request body")},
		Action: func(_ context.Context, cmd *cli.Command) error { return Output(cmd, response) },
	}
	require.NoError(t, cmd.Run(t.Context(), append([]string{"test"}, args...)))
	os.Stdout = original
	output, err := os.ReadFile(file.Name())
	require.NoError(t, err)
	return string(output)
}
