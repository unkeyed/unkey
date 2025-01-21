package middleware

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/unkeyed/unkey/go/pkg/api"
	"github.com/unkeyed/unkey/go/pkg/api/session"
	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/go/pkg/logging"
)

type RequestLogger[TRequest session.Redacter, TResponse session.Redacter] struct {
	next        session.Handler[TRequest, TResponse]
	eventBuffer api.EventBuffer
	logger      logging.Logger
}

var _ session.Handler[session.Redacter, session.Redacter] = (*RequestLogger[session.Redacter, session.Redacter])(nil)

func (mw *RequestLogger[TRequest, TResponse]) Handle(sess session.Session[TRequest, TResponse]) error {

	start := time.Now()
	ctx := sess.Context()

	nextErr := mw.next.Handle(sess)
	serviceLatency := time.Since(start)

	summary, err := sess.Summary()
	if err != nil {
		return err
	}

	mw.logger.Info(ctx, "request",
		slog.String("method", summary.Method),
		slog.String("path", summary.Path),
		slog.Int("status", summary.ResponseStatus),
		slog.String("latency", serviceLatency.String()))

	requestHeaders := []string{}
	for k, vv := range summary.RequestHeader {
		requestHeaders = append(requestHeaders, fmt.Sprintf("%s: %s", k, strings.Join(vv, ",")))
	}

	responseHeaders := []string{}
	for k, vv := range summary.ResponseHeader {
		responseHeaders = append(responseHeaders, fmt.Sprintf("%s: %s", k, strings.Join(vv, ",")))
	}

	mw.eventBuffer.BufferApiRequest(schema.ApiRequestV1{
		RequestID:       sess.RequestID(),
		Time:            start.UnixMilli(),
		Host:            summary.Host,
		Method:          summary.Method,
		Path:            summary.Path,
		RequestHeaders:  requestHeaders,
		RequestBody:     string(summary.RequestBody),
		ResponseStatus:  summary.ResponseStatus,
		ResponseHeaders: responseHeaders,
		ResponseBody:    string(summary.ResponseBody),
		Error:           "",
	})
	return nextErr
}
