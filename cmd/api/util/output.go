package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/pkg/cli"
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
	if rows, ok := envelope.Data.([]any); ok || (envelope.Data == nil && columns != nil) {
		if len(rows) == 0 {
			fmt.Fprintf(&output, "No %s found.\n", subject)
		} else if err := writeTable(&output, rows, columns, "", format); err != nil {
			return err
		}
	} else if envelope.Data == nil || emptyObject(envelope.Data) {
		fmt.Fprintln(&output, successMessage(v))
	} else if err := writeDetails(&output, envelope.Data, "", false); err != nil {
		return err
	}
	if envelope.Pagination != nil && envelope.Pagination.HasMore {
		if cursor := envelope.Pagination.GetCursor(); cursor != nil && *cursor != "" {
			if cmd.FlagIsSet("body") {
				encoded, err := json.Marshal(*cursor)
				if err != nil {
					return err
				}
				fmt.Fprint(&output, "\nMore results available. In --body, set cursor to ")
				for _, codeUnit := range utf16.Encode([]rune(string(encoded))) {
					if codeUnit >= 0x7f {
						fmt.Fprintf(&output, `\u%04x`, codeUnit)
					} else {
						output.WriteByte(byte(codeUnit))
					}
				}
				output.WriteByte('\n')
			} else {
				fmt.Fprintf(&output, "\nMore results available. Repeat this command with --cursor=%s\n", shellQuote(*cursor))
			}
		} else {
			fmt.Fprintln(&output, "\nMore results available, but the API returned no cursor.")
		}
	}
	_, err = fmt.Fprint(os.Stdout, output.String())
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
	default:
		return "results", nil
	}
}

func successMessage(response any) string {
	switch response.(type) {
	case *components.V2ProjectsDeleteProjectResponseBody, *components.V2AppsDeleteAppResponseBody:
		return "Deletion requested."
	case *components.V2DeploymentsPromoteDeploymentResponseBody:
		return "Promotion requested."
	case *components.V2DeploymentsRollbackDeploymentResponseBody:
		return "Rollback requested."
	case *components.V2DeploymentsStartDeploymentResponseBody:
		return "Start requested."
	case *components.V2DeploymentsStopDeploymentResponseBody:
		return "Stop requested."
	case *components.V2DomainsVerifyDomainResponseBody:
		return "Verification requested."
	case *components.V2ApisDeleteAPIResponseBody:
		return "API deleted."
	case *components.V2KeysDeleteKeyResponseBody:
		return "Key deleted."
	case *components.V2KeysUpdateKeyResponseBody:
		return "Key updated."
	case *components.V2IdentitiesDeleteIdentityResponseBody:
		return "Identity deleted."
	case *components.V2PermissionsDeletePermissionResponseBody:
		return "Permission deleted."
	case *components.V2PermissionsDeleteRoleResponseBody:
		return "Role deleted."
	case *components.V2PortalDeletePortalResponseBody:
		return "Portal deleted."
	case *components.V2DomainsDeleteDomainResponseBody:
		return "Domain deleted."
	case *components.V2EnvironmentsUpdateSettingsResponseBody:
		return "Settings updated."
	case *components.V2EnvironmentsSetEnvironmentVariablesResponseBody:
		return "Variables saved."
	case *components.V2EnvironmentsRemoveEnvironmentVariablesResponseBody:
		return "Variables removed."
	case *components.V2GatewaySetPoliciesResponseBody:
		return "Policies updated."
	case *components.V2GatewayUpdatePolicyResponseBody:
		return "Policy updated."
	default:
		return "Done."
	}
}

func emptyObject(value any) bool {
	object, ok := value.(map[string]any)
	return ok && len(object) == 0
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(terminalText(value), "'", "'\"'\"'") + "'"
}
