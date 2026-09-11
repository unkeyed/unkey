package logdrains

import (
	"context"
	"strings"

	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/gen/rpc/vault"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/ssrf"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"golang.org/x/net/http/httpguts"
)

func setStream(config *logdrainv1.Config, stream string, filters openapi.LogdrainFilters) error {
	if (stream != "audit_logs" && filters.EventTypes != nil) ||
		(stream != "key_verifications" && (filters.Outcomes != nil || filters.KeySpaceIds != nil)) ||
		(stream != "ratelimits" && (filters.NamespaceIds != nil || filters.Passed != nil)) ||
		(stream != "gateway_requests" && filters.StatusClasses != nil) ||
		(stream != "runtime_logs" && filters.Severities != nil) ||
		(stream != "gateway_requests" && stream != "runtime_logs" && (filters.ProjectIds != nil || filters.AppIds != nil || filters.EnvironmentIds != nil)) {
		return invalid("Filters must match the drain stream.")
	}
	for _, values := range []*[]string{filters.EventTypes, filters.KeySpaceIds, filters.NamespaceIds, filters.Severities, filters.ProjectIds, filters.AppIds, filters.EnvironmentIds} {
		if values == nil {
			continue
		}
		for i, value := range *values {
			(*values)[i] = strings.TrimSpace(value)
			if (*values)[i] == "" {
				return invalid("Filter values must not be empty.")
			}
		}
	}
	switch stream {
	case "audit_logs":
		config.Stream = &logdrainv1.Config_AuditLogs{AuditLogs: &logdrainv1.AuditLogStreamConfig{EventTypes: ptr.SafeDeref(filters.EventTypes, config.GetAuditLogs().GetEventTypes())}}
	case "key_verifications":
		config.Stream = &logdrainv1.Config_KeyVerifications{KeyVerifications: &logdrainv1.KeyVerificationStreamConfig{Outcomes: ptr.SafeDeref(filters.Outcomes, config.GetKeyVerifications().GetOutcomes()), KeySpaceIds: ptr.SafeDeref(filters.KeySpaceIds, config.GetKeyVerifications().GetKeySpaceIds())}}
	case "ratelimits":
		config.Stream = &logdrainv1.Config_Ratelimits{Ratelimits: &logdrainv1.RatelimitStreamConfig{NamespaceIds: ptr.SafeDeref(filters.NamespaceIds, config.GetRatelimits().GetNamespaceIds()), Passed: ptr.SafeDeref(filters.Passed, config.GetRatelimits().GetPassed())}}
	case "runtime_logs":
		current := config.GetRuntimeLogs()
		config.Stream = &logdrainv1.Config_RuntimeLogs{RuntimeLogs: &logdrainv1.RuntimeLogStreamConfig{Severities: ptr.SafeDeref(filters.Severities, current.GetSeverities()), ProjectIds: ptr.SafeDeref(filters.ProjectIds, current.GetProjectIds()), AppIds: ptr.SafeDeref(filters.AppIds, current.GetAppIds()), EnvironmentIds: ptr.SafeDeref(filters.EnvironmentIds, current.GetEnvironmentIds())}}
	case "gateway_requests":
		current := config.GetGatewayRequests()
		classes := current.GetStatusClasses()
		if filters.StatusClasses != nil {
			classes = make([]logdrainv1.HttpStatusClass, len(*filters.StatusClasses))
			for i, class := range *filters.StatusClasses {
				classes[i] = logdrainv1.HttpStatusClass(class)
			}
		}
		config.Stream = &logdrainv1.Config_GatewayRequests{GatewayRequests: &logdrainv1.GatewayRequestStreamConfig{StatusClasses: classes, ProjectIds: ptr.SafeDeref(filters.ProjectIds, current.GetProjectIds()), AppIds: ptr.SafeDeref(filters.AppIds, current.GetAppIds()), EnvironmentIds: ptr.SafeDeref(filters.EnvironmentIds, current.GetEnvironmentIds())}}
	default:
		return invalid("Unsupported log drain stream.")
	}
	return nil
}

func setDestination(ctx context.Context, client vault.VaultServiceClient, workspaceID string, config *logdrainv1.Config, destination openapi.LogdrainDestinationWrite) error {
	if (destination.Http == nil) == (destination.Axiom == nil) {
		return invalid("Provide exactly one destination.")
	}
	if input := destination.Axiom; input != nil {
		if config.Destination != nil && config.GetAxiom() == nil {
			return invalid("Destination kind cannot be changed. Create a new log drain instead.")
		}
		current := config.GetAxiom()
		if current == nil {
			current = &logdrainv1.AxiomConfig{}
		}
		if input.Dataset != nil {
			current.Dataset = *input.Dataset
		}
		if input.Token != nil {
			encrypted, err := client.Encrypt(ctx, &vaultv1.EncryptRequest{Keyring: workspaceID, Data: *input.Token})
			if err != nil {
				return err
			}
			current.EncryptedToken = encrypted.GetEncrypted()
		}
		if current.Dataset == "" || current.EncryptedToken == "" {
			return invalid("Axiom requires a dataset and token.")
		}
		config.Destination = &logdrainv1.Config_Axiom{Axiom: current}
		return nil
	}
	input := destination.Http
	if config.Destination != nil && config.GetHttp() == nil {
		return invalid("Destination kind cannot be changed. Create a new log drain instead.")
	}
	current := config.GetHttp()
	if current == nil {
		current = &logdrainv1.HttpConfig{}
	}
	if input.Url != nil {
		current.Url = *input.Url
	}
	if err := ssrf.ValidateEndpoint(current.Url); err != nil {
		return invalid("URL must use HTTPS and must not contain credentials.")
	}
	if input.Format != nil {
		switch *input.Format {
		case openapi.LogdrainHttpWriteFormatJson:
			current.Format = logdrainv1.HttpBodyFormat_HTTP_BODY_FORMAT_JSON
		case openapi.LogdrainHttpWriteFormatNdjson:
			current.Format = logdrainv1.HttpBodyFormat_HTTP_BODY_FORMAT_NDJSON
		default:
			return invalid("Unsupported HTTP body format.")
		}
	}
	if input.Headers != nil {
		headers := make([]*logdrainv1.HttpHeader, 0, len(*input.Headers))
		seen := map[string]bool{}
		for _, header := range *input.Headers {
			name := strings.ToLower(header.Name)
			if seen[name] || !httpguts.ValidHeaderFieldName(header.Name) {
				return invalid("HTTP header names must be valid and unique, ignoring case.")
			}
			seen[name] = true
			switch header.Mode {
			case openapi.LogdrainHeaderSet:
				if header.Value == nil || !httpguts.ValidHeaderFieldValue(*header.Value) {
					return invalid("A valid HTTP header value is required for set.")
				}
				encrypted, err := client.Encrypt(ctx, &vaultv1.EncryptRequest{Keyring: workspaceID, Data: *header.Value})
				if err != nil {
					return err
				}
				headers = append(headers, &logdrainv1.HttpHeader{Name: header.Name, EncryptedValue: encrypted.GetEncrypted()})
			case openapi.LogdrainHeaderPreserve:
				if header.Value != nil {
					return invalid("Preserved headers must omit value.")
				}
				var found *logdrainv1.HttpHeader
				for _, existing := range current.Headers {
					if strings.EqualFold(existing.Name, header.Name) {
						found = existing
						break
					}
				}
				if found == nil {
					return invalid("Cannot preserve an unknown HTTP header.")
				}
				headers = append(headers, found)
			default:
				return invalid("Unsupported HTTP header mode.")
			}
		}
		current.Headers = headers
	}
	config.Destination = &logdrainv1.Config_Http{Http: current}
	return nil
}
