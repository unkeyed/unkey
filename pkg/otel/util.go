package otel

import (
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// RecordError marks a span as having encountered an error.
// If the error is nil, this function does nothing.
// This should be called whenever an error occurs within a traced operation
// to ensure that errors are properly recorded in the tracing system.
//
// Example:
//
//	ctx, span := tracing.Start(ctx, "database.Query")
//	defer span.End()
//
//	result, err := db.Query(ctx, "SELECT * FROM users WHERE id = ?", userID)
//	if err != nil {
//	    otel.RecordError(span, err)
//	    return nil, err
//	}
func RecordError(span trace.Span, err error) {
	if err == nil {
		return
	}
	span.SetStatus(codes.Error, err.Error())
}
