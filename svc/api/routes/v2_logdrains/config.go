package logdrains

import (
	"fmt"

	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"google.golang.org/protobuf/proto"
)

func toPublic(row db.Logdrain) (openapi.Logdrain, error) {
	var data openapi.Logdrain
	config := &logdrainv1.Config{}
	if err := proto.Unmarshal(row.Config, config); err != nil {
		return data, fmt.Errorf("decode log drain: %w", err)
	}
	data.Id = row.ID
	data.Name = row.Name
	data.Status = openapi.LogdrainStatus(row.Status)
	data.Stream = openapi.LogdrainStream(row.Stream)
	data.BatchSize = int64(config.GetBatchSize())
	if data.BatchSize == 0 {
		data.BatchSize = 10_000
	}
	data.ConsecutiveFailures = int(row.ConsecutiveFailures)
	data.CommittedOffsetInsertedAt = row.CommittedOffsetInsertedAt
	data.CreatedAt = row.CreatedAt
	switch stream := config.Stream.(type) {
	case nil:
		data.Filters.EventTypes = ptr.P([]string{})
	case *logdrainv1.Config_AuditLogs:
		data.Filters.EventTypes = array(stream.AuditLogs.EventTypes)
	case *logdrainv1.Config_KeyVerifications:
		data.Filters.Outcomes = array(stream.KeyVerifications.Outcomes)
		data.Filters.KeySpaceIds = array(stream.KeyVerifications.KeySpaceIds)
	case *logdrainv1.Config_Ratelimits:
		data.Filters.NamespaceIds = array(stream.Ratelimits.NamespaceIds)
		data.Filters.Passed = array(stream.Ratelimits.Passed)
	case *logdrainv1.Config_RuntimeLogs:
		data.Filters.Severities = array(stream.RuntimeLogs.Severities)
		data.Filters.ProjectIds = array(stream.RuntimeLogs.ProjectIds)
		data.Filters.AppIds = array(stream.RuntimeLogs.AppIds)
		data.Filters.EnvironmentIds = array(stream.RuntimeLogs.EnvironmentIds)
	case *logdrainv1.Config_GatewayRequests:
		classes := make([]openapi.LogdrainFiltersStatusClasses, len(stream.GatewayRequests.StatusClasses))
		for i, class := range stream.GatewayRequests.StatusClasses {
			classes[i] = openapi.LogdrainFiltersStatusClasses(class)
		}
		data.Filters.StatusClasses = &classes
		data.Filters.ProjectIds = array(stream.GatewayRequests.ProjectIds)
		data.Filters.AppIds = array(stream.GatewayRequests.AppIds)
		data.Filters.EnvironmentIds = array(stream.GatewayRequests.EnvironmentIds)
	default:
		return data, fmt.Errorf("unsupported log drain stream %T", stream)
	}
	switch destination := config.Destination.(type) {
	case *logdrainv1.Config_Http:
		format := openapi.LogdrainDestinationHttpFormat("json")
		switch destination.Http.Format {
		case logdrainv1.HttpBodyFormat_HTTP_BODY_FORMAT_UNSPECIFIED, logdrainv1.HttpBodyFormat_HTTP_BODY_FORMAT_JSON:
		case logdrainv1.HttpBodyFormat_HTTP_BODY_FORMAT_NDJSON:
			format = "ndjson"
		default:
			return data, fmt.Errorf("unsupported log drain body format %d", destination.Http.Format)
		}
		headers := make([]string, len(destination.Http.Headers))
		for i, header := range destination.Http.Headers {
			headers[i] = header.Name
		}
		data.Destination.Http = &struct {
			Format  openapi.LogdrainDestinationHttpFormat `json:"format"`
			Headers []string                              `json:"headers"`
			Url     string                                `json:"url"`
		}{Format: format, Headers: headers, Url: destination.Http.Url}
	case *logdrainv1.Config_Axiom:
		data.Destination.Axiom = &struct {
			Dataset string `json:"dataset"`
		}{Dataset: destination.Axiom.Dataset}
	default:
		return data, fmt.Errorf("unsupported log drain destination %T", destination)
	}
	return data, nil
}

func array[T any](values []T) *[]T {
	if values == nil {
		values = []T{}
	}
	return &values
}
