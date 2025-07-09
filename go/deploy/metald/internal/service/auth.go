package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel/baggage"
)

// CustomerContext holds customer information extracted from authentication
type CustomerContext struct {
	CustomerID  string
	TenantID    string
	UserID      string
	WorkspaceID string
}

// AuthenticationInterceptor validates API requests and enforces customer isolation
func AuthenticationInterceptor(logger *slog.Logger) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			// Extract API key from Authorization header
			auth := req.Header().Get("Authorization")
			if auth == "" {
				logger.LogAttrs(ctx, slog.LevelWarn, "missing authorization header",
					slog.String("procedure", req.Spec().Procedure),
				)
				return nil, connect.NewError(connect.CodeUnauthenticated,
					fmt.Errorf("authorization header required"))
			}

			// Parse Bearer token
			parts := strings.SplitN(auth, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				logger.LogAttrs(ctx, slog.LevelWarn, "invalid authorization format",
					slog.String("procedure", req.Spec().Procedure),
				)
				return nil, connect.NewError(connect.CodeUnauthenticated,
					fmt.Errorf("authorization must be 'Bearer <token>'"))
			}

			token := parts[1]

			// Validate token and extract customer context
			customerCtx, err := validateToken(ctx, token)
			if err != nil {
				logger.LogAttrs(ctx, slog.LevelWarn, "token validation failed",
					slog.String("procedure", req.Spec().Procedure),
					slog.String("error", err.Error()),
				)
				return nil, connect.NewError(connect.CodeUnauthenticated, err)
			}

			// Extract requested tenant ID from header and validate access
			requestedTenantID := req.Header().Get("X-Tenant-ID")
			logger.LogAttrs(ctx, slog.LevelInfo, "checking tenant access",
				slog.String("procedure", req.Spec().Procedure),
				slog.String("user_id", customerCtx.CustomerID),
				slog.String("requested_tenant", requestedTenantID),
			)

			if requestedTenantID != "" {
				// Validate that authenticated user can access the requested tenant
				if err := validateTenantAccess(ctx, customerCtx, requestedTenantID); err != nil {
					logger.LogAttrs(ctx, slog.LevelWarn, "tenant access denied",
						slog.String("procedure", req.Spec().Procedure),
						slog.String("user_id", customerCtx.CustomerID),
						slog.String("requested_tenant", requestedTenantID),
						slog.String("error", err.Error()),
					)
					return nil, connect.NewError(connect.CodePermissionDenied, err)
				}
				logger.LogAttrs(ctx, slog.LevelInfo, "tenant access granted",
					slog.String("procedure", req.Spec().Procedure),
					slog.String("user_id", customerCtx.CustomerID),
					slog.String("requested_tenant", requestedTenantID),
				)
			}

			// Add customer context to baggage for downstream services
			ctx = addCustomerContextToBaggage(ctx, customerCtx)

			// Log authenticated request
			logger.LogAttrs(ctx, slog.LevelDebug, "authenticated request",
				slog.String("procedure", req.Spec().Procedure),
				slog.String("customer_id", customerCtx.CustomerID),
				slog.String("tenant_id", customerCtx.TenantID),
			)

			return next(ctx, req)
		}
	}
}

// validateToken validates the API token and returns customer context
// TODO: Replace with your actual authentication mechanism (JWT, API keys, etc.)
func validateToken(ctx context.Context, token string) (*CustomerContext, error) {
	_ = ctx // Will be used for auth service calls in production
	// DEVELOPMENT MODE: Extract customer_id from token directly
	// Format: "dev_customer_<customer_id>" for development
	// Production should validate against your auth service

	// Development mode: Accept simple bearer tokens
	if strings.HasPrefix(token, "dev_user_") {
		userID := strings.TrimPrefix(token, "dev_user_")
		if userID == "" {
			return nil, fmt.Errorf("invalid development token format")
		}

		return &CustomerContext{
			CustomerID:  userID,
			TenantID:    "", // Tenant determined by X-Tenant-ID header
			UserID:      userID,
			WorkspaceID: "dev_workspace",
		}, nil
	}

	// Legacy support for old dev_customer_ format
	if strings.HasPrefix(token, "dev_customer_") {
		customerID := strings.TrimPrefix(token, "dev_customer_")
		if customerID == "" {
			return nil, fmt.Errorf("invalid development token format")
		}

		return &CustomerContext{
			CustomerID:  customerID,
			TenantID:    customerID, // Use customer ID as tenant ID for legacy
			UserID:      customerID,
			WorkspaceID: "dev_workspace",
		}, nil
	}

	// Production token validation would go here
	// Example: JWT validation, API key lookup, etc.
	return nil, fmt.Errorf("invalid token format - use 'dev_user_<user_id>' for development")
}

// addCustomerContextToBaggage adds customer context to OpenTelemetry baggage
func addCustomerContextToBaggage(ctx context.Context, customerCtx *CustomerContext) context.Context {
	// Create baggage with customer context
	bag, err := baggage.Parse(fmt.Sprintf(
		"customer_id=%s,tenant_id=%s,user_id=%s,workspace_id=%s",
		customerCtx.CustomerID,
		customerCtx.TenantID,
		customerCtx.UserID,
		customerCtx.WorkspaceID,
	))
	if err != nil {
		// Log error but continue - baggage is for observability, not security
		slog.Default().WarnContext(ctx, "failed to create baggage",
			slog.String("error", err.Error()),
		)
		return ctx
	}

	return baggage.ContextWithBaggage(ctx, bag)
}

// ExtractCustomerID extracts customer ID from request context
func ExtractCustomerID(ctx context.Context) (string, error) {
	if requestBaggage := baggage.FromContext(ctx); len(requestBaggage.Members()) > 0 {
		customerID := requestBaggage.Member("customer_id").Value()
		if customerID != "" {
			return customerID, nil
		}
	}
	return "", fmt.Errorf("customer_id not found in context")
}

// validateTenantAccess validates that the authenticated user can access the requested tenant
func validateTenantAccess(ctx context.Context, customerCtx *CustomerContext, requestedTenantID string) error {
	// AIDEV-BUSINESS_RULE: Tenant access validation for multi-tenant security

	// In development mode, allow any authenticated user to access any tenant
	// TODO: In production, implement proper tenant-user relationship checks
	// This should query a tenant membership service or database

	// For now, basic validation that tenant ID is not empty
	if requestedTenantID == "" {
		return fmt.Errorf("tenant ID cannot be empty")
	}

	// Development: Simple access control for demonstration
	// Block access to "restricted-tenant" unless user is "admin-user"
	if requestedTenantID == "restricted-tenant" && customerCtx.CustomerID != "admin-user" {
		return fmt.Errorf("access denied: user %s cannot access restricted tenant", customerCtx.CustomerID)
	}

	// In production, this would check:
	// 1. User has permission to access the tenant
	// 2. User's role within the tenant (admin, user, etc.)
	// 3. Specific resource permissions if needed

	// Example future implementation:
	// tenantService := GetTenantServiceFromContext(ctx)
	// return tenantService.ValidateUserAccess(customerCtx.CustomerID, requestedTenantID)

	return nil // Allow all other access in development
}

// validateVMOwnership validates that the customer owns the specified VM
func (s *VMService) validateVMOwnership(ctx context.Context, vmID string) error {
	// Extract customer ID from authenticated context
	customerID, err := ExtractCustomerID(ctx)
	if err != nil {
		return connect.NewError(connect.CodeUnauthenticated, err)
	}

	// Get VM from database
	vm, err := s.vmRepo.GetVMWithContext(ctx, vmID)
	if err != nil {
		s.logger.LogAttrs(ctx, slog.LevelWarn, "vm not found during ownership validation",
			slog.String("vm_id", vmID),
			slog.String("customer_id", customerID),
		)
		return connect.NewError(connect.CodeNotFound, fmt.Errorf("VM not found: %s", vmID))
	}

	// Validate ownership
	if vm.CustomerID != customerID {
		s.logger.LogAttrs(ctx, slog.LevelWarn, "SECURITY: unauthorized vm access attempt",
			slog.String("vm_id", vmID),
			slog.String("requesting_customer", customerID),
			slog.String("vm_owner", vm.CustomerID),
			slog.String("action", "access_denied"),
		)
		return connect.NewError(connect.CodePermissionDenied,
			fmt.Errorf("access denied: VM not owned by customer"))
	}

	return nil
}
