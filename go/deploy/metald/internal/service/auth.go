package service

// import (
// 	"context"
// 	"fmt"
// 	"log/slog"
// 	"strings"

// 	"connectrpc.com/connect"
// 	"go.opentelemetry.io/otel/baggage"
// )

// // CustomerContext holds customer information extracted from authentication
// type CustomerContext struct {
// 	UserID        string
// 	TenantID      string
// 	ProjectID     string
// 	EnvironmentID string
// }

// // AuthenticationInterceptor validates API requests and enforces customer isolation
// func AuthenticationInterceptor(logger *slog.Logger) connect.UnaryInterceptorFunc {
// 	return func(next connect.UnaryFunc) connect.UnaryFunc {
// 		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
// 			// Extract API key from Authorization header
// 			auth := req.Header().Get("Authorization")
// 			if auth == "" {
// 				logger.LogAttrs(ctx, slog.LevelWarn, "missing authorization header",
// 					slog.String("procedure", req.Spec().Procedure),
// 				)
// 				return nil, connect.NewError(connect.CodeUnauthenticated,
// 					fmt.Errorf("authorization header required"))
// 			}

// 			// Parse Bearer token
// 			parts := strings.SplitN(auth, " ", 2)
// 			if len(parts) != 2 || parts[0] != "Bearer" {
// 				logger.LogAttrs(ctx, slog.LevelWarn, "invalid authorization format",
// 					slog.String("procedure", req.Spec().Procedure),
// 				)
// 				return nil, connect.NewError(connect.CodeUnauthenticated,
// 					fmt.Errorf("authorization must be 'Bearer <token>'"))
// 			}

// 			// Extract requested tenant ID from header and validate access
// 			requestedTenantID := req.Header().Get("X-Tenant-ID")
// 			requestedProjectID := req.Header().Get("X-Project-ID")
// 			requestedEnvironmentID := req.Header().Get("X-Environment-ID")
// 			logger.LogAttrs(ctx, slog.LevelInfo, "checking tenant access",
// 				slog.String("procedure", req.Spec().Procedure),
// 				slog.String("user_id", customerCtx.UserID),
// 				slog.String("requested_tenant", requestedTenantID),
// 				slog.String("requested_project", requestedProjectID),
// 				slog.String("requested_environment", requestedEnvironmentID),
// 			)

// 			if requestedTenantID != "" {
// 				// Validate that authenticated user can access the requested tenant
// 				if err := validateTenantAccess(ctx, customerCtx, requestedTenantID); err != nil {
// 					logger.LogAttrs(ctx, slog.LevelWarn, "tenant access denied",
// 						slog.String("procedure", req.Spec().Procedure),
// 						slog.String("user_id", customerCtx.UserID),
// 						slog.String("requested_tenant", requestedTenantID),
// 						slog.String("error", err.Error()),
// 					)
// 					return nil, connect.NewError(connect.CodePermissionDenied, err)
// 				}
// 				logger.LogAttrs(ctx, slog.LevelInfo, "tenant access granted",
// 					slog.String("procedure", req.Spec().Procedure),
// 					slog.String("user_id", customerCtx.UserID),
// 					slog.String("requested_tenant", requestedTenantID),
// 				)
// 			}

// 			// Add customer context to baggage for downstream services
// 			ctx = addCustomerContextToBaggage(ctx, customerCtx)

// 			// Log authenticated request
// 			logger.LogAttrs(ctx, slog.LevelDebug, "authenticated request",
// 				slog.String("procedure", req.Spec().Procedure),
// 				slog.String("user_id", customerCtx.UserID),
// 				slog.String("tenant_id", customerCtx.TenantID),
// 			)

// 			return next(ctx, req)
// 		}
// 	}
// }

// // addCustomerContextToBaggage adds customer context to OpenTelemetry baggage
// func addCustomerContextToBaggage(ctx context.Context, customerCtx *CustomerContext) context.Context {
// 	// Create baggage with customer context
// 	bag, err := baggage.Parse(fmt.Sprintf(
// 		"user_id=%s,project_id=%s,tenant_id=%s,environment_id=%s",
// 		customerCtx.UserID,
// 		customerCtx.ProjectID,
// 		customerCtx.TenantID,
// 		customerCtx.EnvironmentID,
// 	))
// 	if err != nil {
// 		// Log error but continue - baggage is for observability, not security
// 		slog.Default().WarnContext(ctx, "failed to create baggage",
// 			slog.String("error", err.Error()),
// 		)
// 		return ctx
// 	}

// 	return baggage.ContextWithBaggage(ctx, bag)
// }

// // ExtractTenantID extracts tenant ID from request context
// func ExtractTenantID(ctx context.Context) (string, error) {
// 	if requestBaggage := baggage.FromContext(ctx); len(requestBaggage.Members()) > 0 {
// 		tenantID := requestBaggage.Member("tenant_id").Value()
// 		if tenantID != "" {
// 			return tenantID, nil
// 		}
// 	}
// 	return "", fmt.Errorf("tenant_id not found in context")
// }

// // ExtractEnvironmentID extracts tenant ID from request context
// func ExtractEnvironmentID(ctx context.Context) (string, error) {
// 	if requestBaggage := baggage.FromContext(ctx); len(requestBaggage.Members()) > 0 {
// 		environmentID := requestBaggage.Member("environment_id").Value()
// 		if environmentID != "" {
// 			return environmentID, nil
// 		}
// 	}
// 	return "", fmt.Errorf("environment_id not found in context")
// }

// // ExtractProjectID extracts project ID from request context
// func ExtractProjectID(ctx context.Context) (string, error) {
// 	if requestBaggage := baggage.FromContext(ctx); len(requestBaggage.Members()) > 0 {
// 		projectID := requestBaggage.Member("project_id").Value()
// 		if projectID != "" {
// 			return projectID, nil
// 		}
// 	}
// 	return "", fmt.Errorf("project_id not found in context")
// }

// // ExtractUserID extracts tenant ID from request context
// func ExtractUserID(ctx context.Context) (string, error) {
// 	if requestBaggage := baggage.FromContext(ctx); len(requestBaggage.Members()) > 0 {
// 		userID := requestBaggage.Member("user_id").Value()
// 		if userID != "" {
// 			return userID, nil
// 		}
// 	}
// 	return "", fmt.Errorf("tenant_id not found in context")
// }

// // validateTenantAccess validates that the authenticated user can access the requested tenant
// func validateTenantAccess(ctx context.Context, customerCtx *CustomerContext, requestedTenantID string) error {
// 	// AIDEV-BUSINESS_RULE: Tenant access validation for multi-tenant security

// 	// In development mode, allow any authenticated user to access any tenant
// 	// TODO: In production, implement proper tenant-user relationship checks
// 	// This should query a tenant membership service or database using ctx for timeouts/tracing
// 	_ = ctx // Will be used for database queries in production implementation

// 	// For now, basic validation that tenant ID is not empty
// 	if requestedTenantID == "" {
// 		return fmt.Errorf("tenant ID cannot be empty")
// 	}

// 	// Development: Simple access control for demonstration
// 	// Block access to "restricted-tenant" unless user is "admin-user"
// 	if requestedTenantID == "restricted-tenant" && customerCtx.UserID != "admin-user" {
// 		return fmt.Errorf("access denied: user %s cannot access restricted tenant", customerCtx.UserID)
// 	}

// 	// In production, this would check:
// 	// 1. User has permission to access the tenant
// 	// 2. User's role within the tenant (admin, user, etc.)
// 	// 3. Specific resource permissions if needed

// 	// Example future implementation:
// 	// tenantService := GetTenantServiceFromContext(ctx)
// 	// return tenantService.ValidateUserAccess(customerCtx.CustomerID, requestedTenantID)

// 	return nil // Allow all other access in development
// }
