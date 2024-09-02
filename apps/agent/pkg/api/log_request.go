package api

import (
	"context"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/unkeyed/unkey/apps/agent/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
)

type Redactor interface {
	Redact() (headers, body string, err error)
}

func (s *Server) LogRequest(ctx context.Context, op huma.Operation, req Redactor, res Redactor, err error) {
	ctx, span := tracing.Start(ctx, tracing.NewSpanName("log", op.OperationID))
	defer span.End()

	requestID, ok := ctx.Value("requestID").(string)
	if !ok {
		s.logger.Error().Msg("requestID not found in context")
		requestID = ""
	}
	host, ok := ctx.Value("host").(string)
	if !ok {
		s.logger.Error().Msg("host not found in context")
		requestID = ""
	}
	method, ok := ctx.Value("method").(string)
	if !ok {
		s.logger.Error().Msg("method not found in context")
		requestID = ""
	}
	path, ok := ctx.Value("path").(string)
	if !ok {
		s.logger.Error().Msg("path not found in context")
		requestID = ""
	}

	requestHeaders, requestBody, err := req.Redact()
	if err != nil {
		tracing.RecordError(span, err)
		s.logger.Error().Err(err).Msg("failed to redact request")
		requestHeaders = ""
		requestBody = ""
	}
	responseHeaders, responseBody, err := res.Redact()
	if err != nil {
		tracing.RecordError(span, err)
		s.logger.Error().Err(err).Msg("failed to redact response")
		responseHeaders = ""
		responseBody = ""
	}

	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	s.clickhouse.BufferApiRequest(schema.ApiRequestV1{
		RequestID:       requestID,
		Time:            time.Now().UnixMilli(),
		Host:            host,
		Method:          method,
		Path:            path,
		RequestHeaders:  requestHeaders,
		RequestBody:     requestBody,
		ResponseStatus:  op.DefaultStatus,
		ResponseHeaders: responseHeaders,
		ResponseBody:    responseBody,
		Error:           errStr,
	})
}
