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
	// TenantID is the unique identifier for the tenant/customer.
	TenantID string
	// ProjectID is the unique identifier for the project.
	ProjectID string
	// EnvironmentID is the environment within the project
	EnvironmentID string
	// UserID is the user ID making the request
	UserID string
	// AuthToken is the authentication token provided in the request.
	AuthToken string
}

// contextKey is a private type for context keys to avoid collisions.
type contextKey string

const (
	tenantContextKey      contextKey = "tenant_id"
	projectContextKey     contextKey = "project_id"
	environmentContextKey contextKey = "environment_id"
	userContextKey        contextKey = "user_id"
)

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
			// Panic recovery in tenant auth interceptor prevents auth failures from crashing the service
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
			projectID := req.Header().Get("X-Project-ID")
			environmentID := req.Header().Get("X-Environment-ID")
			userID := req.Header().Get("X-User-ID")
			authToken := req.Header().Get("Authorization")

			// Log request with tenant info if logger is available
			if options.Logger != nil && options.Logger.Enabled(ctx, slog.LevelDebug) {
				options.Logger.LogAttrs(ctx, slog.LevelDebug, "tenant auth headers",
					slog.String("service", options.ServiceName),
					slog.String("procedure", req.Spec().Procedure),
					slog.String("tenant_id", tenantID),
					slog.String("project_id", projectID),
					slog.String("environment_id", environmentID),
					slog.String("user_id", userID),
					slog.Bool("has_auth_token", authToken != ""),
				)
			}

			// Add tenant context to the request context
			tenantCtx := TenantContext{
				TenantID:      tenantID,
				ProjectID:     projectID,
				EnvironmentID: environmentID,
				UserID:        userID,
				AuthToken:     authToken,
			}
			ctx = WithTenantContext(ctx, tenantCtx)

			// Add tenant info to span if tracing is enabled
			if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
				span.SetAttributes(
					attribute.String("tenant.id", tenantID),
					attribute.String("tenant.project_id", projectID),
					attribute.String("tenant.environment_id", environmentID),
					attribute.String("user.id", userID),
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

			// THIS IS PROBABLY WHERE UNKEY OR SOMETHING DOES AUTH LOL

			// Log successful tenant authentication
			if options.Logger != nil && tenantID != "" {
				options.Logger.LogAttrs(ctx, slog.LevelDebug, "tenant authenticated",
					slog.String("service", options.ServiceName),
					slog.String("tenant_id", tenantID),
					slog.String("project_id", projectID),
					slog.String("environment_id", environmentID),
					slog.String("user_id", userID),
					slog.String("procedure", req.Spec().Procedure),
				)
			}

			return next(ctx, req)
		}
	}
}
