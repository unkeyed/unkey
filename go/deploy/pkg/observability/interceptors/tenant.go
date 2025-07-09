package interceptors

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// TenantContext holds tenant authentication information extracted from request headers.
type TenantContext struct {
	// TenantID is the unique identifier for the tenant.
	TenantID string
	// CustomerID is the unique identifier for the customer.
	CustomerID string
	// AuthToken is the authentication token provided in the request.
	AuthToken string
}

// contextKey is a private type for context keys to avoid collisions.
type contextKey string

const tenantContextKey contextKey = "tenant_auth"

// WithTenantContext adds tenant authentication context to the context.
func WithTenantContext(ctx context.Context, auth TenantContext) context.Context {
	return context.WithValue(ctx, tenantContextKey, auth)
}

// TenantFromContext extracts tenant authentication context from the context.
// Returns the TenantContext and a boolean indicating if it was found.
func TenantFromContext(ctx context.Context) (TenantContext, bool) {
	auth, ok := ctx.Value(tenantContextKey).(TenantContext)
	return auth, ok
}

// NewTenantAuthInterceptor creates a ConnectRPC interceptor for tenant authentication.
// This interceptor extracts tenant information from request headers, validates it,
// and adds it to the request context for use by downstream handlers.
//
// AIDEV-NOTE: All services need tenant awareness for proper isolation and billing.
func NewTenantAuthInterceptor(opts ...Option) connect.UnaryInterceptorFunc {
	options := applyOptions(opts)

	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (resp connect.AnyResponse, err error) {
			// AIDEV-NOTE: Panic recovery in tenant auth interceptor prevents auth failures from crashing the service
			defer func() {
				if r := recover(); r != nil {
					if options.Logger != nil {
						options.Logger.Error("panic in tenant auth interceptor",
							slog.String("service", options.ServiceName),
							slog.String("procedure", req.Spec().Procedure),
							slog.Any("panic", r),
							slog.String("panic_type", fmt.Sprintf("%T", r)),
						)
					}
					// Only override err if it's not already set
					if err == nil {
						err = connect.NewError(connect.CodeInternal, fmt.Errorf("internal server error: %v", r))
					}
				}
			}()

			// Extract tenant information from headers
			tenantID := req.Header().Get("X-Tenant-ID")
			customerID := req.Header().Get("X-Customer-ID")
			authToken := req.Header().Get("Authorization")

			// Log request with tenant info if logger is available
			if options.Logger != nil && options.Logger.Enabled(ctx, slog.LevelDebug) {
				options.Logger.LogAttrs(ctx, slog.LevelDebug, "tenant auth headers",
					slog.String("service", options.ServiceName),
					slog.String("procedure", req.Spec().Procedure),
					slog.String("tenant_id", tenantID),
					slog.String("customer_id", customerID),
					slog.Bool("has_auth_token", authToken != ""),
				)
			}

			// Add tenant context to the request context
			tenantCtx := TenantContext{
				TenantID:   tenantID,
				CustomerID: customerID,
				AuthToken:  authToken,
			}
			ctx = WithTenantContext(ctx, tenantCtx)

			// Add tenant info to span if tracing is enabled
			if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
				span.SetAttributes(
					attribute.String("tenant.id", tenantID),
					attribute.String("tenant.customer_id", customerID),
					attribute.Bool("tenant.authenticated", tenantID != ""),
				)
			}

			// Check if this procedure requires tenant authentication
			if options.TenantAuthRequired && tenantID == "" {
				// Check if this procedure is exempt from tenant auth
				if !slices.Contains(options.TenantAuthExemptProcedures, req.Spec().Procedure) {
					if options.Logger != nil {
						options.Logger.LogAttrs(ctx, slog.LevelWarn, "missing tenant ID",
							slog.String("service", options.ServiceName),
							slog.String("procedure", req.Spec().Procedure),
						)
					}
					return nil, connect.NewError(connect.CodeUnauthenticated,
						fmt.Errorf("tenant ID is required"))
				}
			}

			// Log successful tenant authentication
			if options.Logger != nil && tenantID != "" {
				options.Logger.LogAttrs(ctx, slog.LevelDebug, "tenant authenticated",
					slog.String("service", options.ServiceName),
					slog.String("tenant_id", tenantID),
					slog.String("customer_id", customerID),
					slog.String("procedure", req.Spec().Procedure),
				)
			}

			// AIDEV-TODO: Add actual token validation logic here when auth service is available
			// This would involve:
			// 1. Validating the auth token with an auth service
			// 2. Checking tenant permissions for the requested procedure
			// 3. Potentially caching validation results for performance

			return next(ctx, req)
		}
	}
}
