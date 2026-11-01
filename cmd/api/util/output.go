package util

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/tui"
	"golang.org/x/term"
)

// Output defaults to human-readable data. JSON preserves the full API envelope;
// debug diagnostics go to stderr, never into piped results.
func Output(cmd *cli.Command, v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}

	var envelope struct {
		Meta       components.Meta        `json:"meta"`
		Data       any                    `json:"data"`
		Pagination *components.Pagination `json:"pagination"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&envelope); err != nil {
		return err
	}
	if cmd.Bool("debug") && envelope.Meta.RequestID != "" {
		if _, err := fmt.Fprintln(os.Stderr, "Request ID:", terminalText(envelope.Meta.RequestID)); err != nil {
			return err
		}
	}
	if cmd.Bool("json") || cmd.Enum("output") == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	}

	format := tableFormat{width: 0, now: time.Now()}
	if term.IsTerminal(int(os.Stdout.Fd())) {
		width, _, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil || width <= 0 {
			width = 80
		}
		format.width = width
	}
	subject, columns := listPresentation(v)
	var output strings.Builder
	message := successMessage(v)
	var request map[string]any
	indent := ""
	if message != "" {
		if cmd.FlagIsSet("body") {
			if err := json.Unmarshal([]byte(cmd.String("body")), &request); err != nil {
				return fmt.Errorf("cannot read mutation target: %w", err)
			}
		}
		if strings.HasPrefix(message, "Created ") {
			if name := cmp.Or(requestValue(cmd, request, "slug", "slug"), requestValue(cmd, request, "name", "name")); name != "" {
				message += " " + terminalText(name)
			}
		}
		fmt.Fprintf(&output, "%s %s\n", tui.New(os.Stdout).Green("✓"), message)
		indent = "  "
	}
	if _, ok := v.(*components.V2KeysWhoamiResponseBody); ok {
		if err := writeKeyTree(&output, envelope.Data, format); err != nil {
			return err
		}
	} else if rows, ok := envelope.Data.([]any); ok || (envelope.Data == nil && columns != nil) {
		if message != "" {
			writeMutationTargets(&output, cmd, request, indent)
		}
		if _, ok := v.(*components.V2EnvironmentsListEnvironmentVariablesResponseBody); ok {
			for _, row := range rows {
				variable, ok := row.(map[string]any)
				if !ok {
					return fmt.Errorf("expected an environment variable, got %T", row)
				}
				if variable["kind"] == "writeonly" {
					variable["value"] = "not retrievable"
				} else if !cmd.Bool("reveal") {
					variable["value"] = "[hidden]"
				} else if variable["value"] == nil || variable["value"] == "" {
					variable["value"] = "(empty)"
				}
			}
		}
		if len(rows) == 0 {
			if message != "" {
				fmt.Fprintf(&output, "%sNo %s assigned.\n", indent, subject)
			} else {
				fmt.Fprintf(&output, "No %s found.\n", subject)
			}
		} else if err := writeTable(&output, rows, columns, indent, format); err != nil {
			return err
		}
	} else if envelope.Data == nil || emptyObject(envelope.Data) {
		if message != "" {
			writeMutationTargets(&output, cmd, request, indent)
		} else if _, ok := v.(*components.V2WorkspaceGetFlagsResponseBody); ok {
			fmt.Fprintln(&output, "No workspace flags found.")
		} else {
			fmt.Fprintln(&output, "Done.")
		}
	} else if err := writeDetails(&output, envelope.Data, indent, false); err != nil {
		return err
	}
	if envelope.Pagination != nil && envelope.Pagination.HasMore {
		if cursor := envelope.Pagination.GetCursor(); cursor != nil && *cursor != "" {
			if cmd.FlagIsSet("body") {
				encoded, err := json.Marshal(*cursor)
				if err != nil {
					return err
				}
				fmt.Fprint(&output, "\nMore results available. Keep the same filters in --body and set cursor to ")
				for _, codeUnit := range utf16.Encode([]rune(string(encoded))) {
					if codeUnit >= 0x7f {
						fmt.Fprintf(&output, `\u%04x`, codeUnit)
					} else {
						output.WriteByte(byte(codeUnit))
					}
				}
				output.WriteByte('\n')
			} else {
				fmt.Fprintf(&output, "\nMore results available. Repeat this command with the same filters and --cursor=%s\n", shellQuote(*cursor))
			}
		} else {
			fmt.Fprintln(&output, "\nMore results available, but the API returned no cursor.")
		}
	}
	if _, err := fmt.Fprint(os.Stdout, output.String()); err != nil {
		return err
	}
	switch v.(type) {
	case *components.V2KeysCreateKeyResponseBody, *components.V2KeysRerollKeyResponseBody:
		_, err = fmt.Fprintln(os.Stderr, "Store this key securely. Only recoverable keys can be retrieved later.")
	case *components.V2RootKeysCreateKeyResponseBody, *components.V2RootKeysRerollKeyResponseBody:
		_, err = fmt.Fprintln(os.Stderr, "Store this root key securely. It cannot be retrieved later.")
	}
	return err
}

func listPresentation(response any) (string, []string) {
	switch response := response.(type) {
	case *components.V2AnalyticsGetGatewayRequestsResponseBody, *components.V2AnalyticsGetRatelimitsResponseBody,
		*components.V2AnalyticsGetRuntimeLogsResponseBody, *components.V2AnalyticsGetVerificationsResponseBody:
		return "results", []string{}
	case *components.V2ProjectsListProjectsResponseBody:
		return "projects", []string{"name", "slug", "updatedAt"}
	case *components.V2AppsListAppsResponseBody:
		return "apps", []string{"name", "slug", "sourceType", "updatedAt"}
	case *components.V2DeploymentsListDeploymentsResponseBody:
		return "deployments", []string{"project", "app", "environment", "status", "isCurrent", "createdAt", "id"}
	case *components.V2DeploymentsListBuildLogsResponseBody:
		return "build logs", []string{"time", "stepId", "step", "output", "message"}
	case *components.V2DomainsListDomainsResponseBody:
		return "domains", []string{"domain", "status", "createdAt"}
	case *components.V2EnvironmentsListEnvironmentsResponseBody:
		return "environments", []string{"slug", "kind", "description"}
	case *components.V2EnvironmentsListEnvironmentVariablesResponseBody:
		return "environment variables", []string{"key", "kind", "value"}
	case *components.V2ApisListKeysResponseBody:
		columns := []string{"name", "start", "enabled", "lastUsedAt", "keyId"}
		for _, key := range response.Data {
			if key.Plaintext != nil {
				columns = append(columns, "plaintext")
				break
			}
		}
		return "keys", columns
	case *components.V2RootKeysListKeysResponseBody:
		return "root keys", []string{"name", "start", "enabled", "expires", "lastUsedAt", "keyId"}
	case *components.V2LogsListAPIRequestsResponseBody:
		return "API requests", []string{"time", "method", "path", "status", "latencyMs", "requestId"}
	case *components.V2LogsGetAPIRequestMetricsResponseBody:
		return "API request metrics", []string{"time", "success", "clientError", "serverError", "total"}
	case *components.V2IdentitiesListIdentitiesResponseBody:
		return "identities", []string{"externalId", "id"}
	case *components.V2PermissionsListPermissionsResponseBody, *components.V2KeysAddPermissionsResponseBody,
		*components.V2KeysRemovePermissionsResponseBody, *components.V2KeysSetPermissionsResponseBody,
		*components.V2PermissionsSetRolePermissionsResponseBody:
		return "permissions", []string{"name", "slug", "description"}
	case *components.V2PermissionsListRolesResponseBody, *components.V2KeysAddRolesResponseBody,
		*components.V2KeysRemoveRolesResponseBody, *components.V2KeysSetRolesResponseBody:
		return "roles", []string{"name", "description"}
	case *components.V2GatewayListPoliciesResponseBody:
		return "policies", []string{"name", "enabled", "id"}
	case *components.V2RatelimitListOverridesResponseBody:
		return "overrides", []string{"identifier", "limit", "duration", "overrideId"}
	case *components.V2RatelimitListNamespacesResponseBody:
		return "rate limit namespaces", []string{"name", "createdAt", "id"}
	default:
		return "results", nil
	}
}

func successMessage(response any) string {
	switch response.(type) {
	case *components.V2ProjectsCreateProjectResponseBody:
		return "Created project"
	case *components.V2ProjectsUpdateProjectResponseBody:
		return "Updated project"
	case *components.V2AppsCreateAppResponseBody:
		return "Created app"
	case *components.V2AppsUpdateAppResponseBody:
		return "Updated app"
	case *components.V2ProjectsDeleteProjectResponseBody, *components.V2AppsDeleteAppResponseBody:
		return "Deletion requested"
	case *components.V2DeploymentsCreateDeploymentResponseBody, *components.V3DeploymentsCreateDeploymentResponseBody:
		return "Deployment requested"
	case *components.V2DeploymentsPromoteDeploymentResponseBody:
		return "Promotion requested"
	case *components.V2DeploymentsRollbackDeploymentResponseBody:
		return "Rollback requested"
	case *components.V2DeploymentsStartDeploymentResponseBody:
		return "Start requested"
	case *components.V2DeploymentsStopDeploymentResponseBody:
		return "Stop requested"
	case *components.V2DeploymentsAuthorizeDeploymentResponseBody:
		return "Authorization requested"
	case *components.V2DeploymentsCancelDeploymentResponseBody:
		return "Cancellation requested"
	case *components.V2DomainsVerifyDomainResponseBody:
		return "Verification requested"
	case *components.V2ApisCreateAPIResponseBody:
		return "Created API"
	case *components.V2ApisDeleteAPIResponseBody:
		return "Deleted API"
	case *components.V2KeysCreateKeyResponseBody:
		return "Created key"
	case *components.V2KeysRerollKeyResponseBody:
		return "Rerolled key"
	case *components.V2KeysDeleteKeyResponseBody:
		return "Deleted key"
	case *components.V2KeysUpdateKeyResponseBody:
		return "Updated key"
	case *components.V2KeysUpdateCreditsResponseBody:
		return "Updated credits"
	case *components.V2KeysAddRolesResponseBody, *components.V2KeysRemoveRolesResponseBody,
		*components.V2KeysSetRolesResponseBody:
		return "Updated roles"
	case *components.V2KeysAddPermissionsResponseBody, *components.V2KeysRemovePermissionsResponseBody,
		*components.V2KeysSetPermissionsResponseBody, *components.V2PermissionsSetRolePermissionsResponseBody:
		return "Updated permissions"
	case *components.V2RootKeysCreateKeyResponseBody:
		return "Created root key"
	case *components.V2RootKeysRerollKeyResponseBody:
		return "Rerolled root key"
	case *components.V2RootKeysDeleteKeyResponseBody:
		return "Deleted root key"
	case *components.V2RootKeysUpdateKeyResponseBody:
		return "Updated root key"
	case *components.V2IdentitiesCreateIdentityResponseBody:
		return "Created identity"
	case *components.V2IdentitiesUpdateIdentityResponseBody:
		return "Updated identity"
	case *components.V2IdentitiesDeleteIdentityResponseBody:
		return "Deleted identity"
	case *components.V2PermissionsCreatePermissionResponseBody:
		return "Created permission"
	case *components.V2PermissionsUpdatePermissionResponseBody:
		return "Updated permission"
	case *components.V2PermissionsDeletePermissionResponseBody:
		return "Deleted permission"
	case *components.V2PermissionsCreateRoleResponseBody:
		return "Created role"
	case *components.V2PermissionsUpdateRoleResponseBody:
		return "Updated role"
	case *components.V2PermissionsDeleteRoleResponseBody:
		return "Deleted role"
	case *components.V2PortalCreatePortalResponseBody:
		return "Created portal"
	case *components.V2PortalUpdatePortalResponseBody:
		return "Updated portal"
	case *components.V2PortalDeletePortalResponseBody:
		return "Deleted portal"
	case *components.V2DomainsCreateDomainResponseBody:
		return "Created domain"
	case *components.V2DomainsDeleteDomainResponseBody:
		return "Deleted domain"
	case *components.V2EnvironmentsUpdateSettingsResponseBody:
		return "Updated settings"
	case *components.V2EnvironmentsSetEnvironmentVariablesResponseBody:
		return "Saved variables"
	case *components.V2EnvironmentsRemoveEnvironmentVariablesResponseBody:
		return "Removed variables"
	case *components.V2GatewaySetPoliciesResponseBody:
		return "Updated policies"
	case *components.V2GatewayUpdatePolicyResponseBody:
		return "Updated policy"
	case *components.V2RatelimitCreateNamespaceResponseBody:
		return "Created rate limit namespace"
	case *components.V2RatelimitUpdateNamespaceResponseBody:
		return "Updated rate limit namespace"
	case *components.V2RatelimitDeleteNamespaceResponseBody:
		return "Deleted rate limit namespace"
	case *components.V2RatelimitSetOverrideResponseBody:
		return "Saved rate limit override"
	case *components.V2RatelimitDeleteOverrideResponseBody:
		return "Deleted rate limit override"
	case *components.V2WorkspaceUpdateFlagResponseBody:
		return "Updated workspace flag"
	default:
		return ""
	}
}

func requestValue(cmd *cli.Command, request map[string]any, field, flag string) string {
	if cmd.FlagIsSet("body") {
		value, _ := request[field].(string)
		return value
	}
	return cmd.String(flag)
}

func writeMutationTargets(output *strings.Builder, cmd *cli.Command, request map[string]any, indent string) {
	for _, selector := range []struct{ field, flag string }{
		{"project", "project"},
		{"app", "app"},
		{"environment", "environment"},
		{"apiId", "api-id"},
		{"keyId", "key-id"},
		{"deploymentId", "deployment-id"},
		{"domain", "domain"},
		{"identity", "identity"},
		{"permission", "permission"},
		{"role", "role"},
		{"portal", "portal"},
		{"namespace", "namespace"},
		{"identifier", "identifier"},
		{"policyId", "policy-id"},
		{"slug", "slug"},
	} {
		if value := requestValue(cmd, request, selector.field, selector.flag); value != "" {
			fmt.Fprintf(output, "%s%s: %s\n", indent, fieldLabel(selector.field), terminalText(value))
		}
	}
}

func emptyObject(value any) bool {
	object, ok := value.(map[string]any)
	return ok && len(object) == 0
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(terminalText(value), "'", "'\"'\"'") + "'"
}
