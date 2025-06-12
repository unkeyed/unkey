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
	// DEVELOPMENT MODE: Extract customer_id from token directly
	// Format: "dev_customer_<customer_id>" for development
	// Production should validate against your auth service
	
	if strings.HasPrefix(token, "dev_customer_") {
		customerID := strings.TrimPrefix(token, "dev_customer_")
		if customerID == "" {
			return nil, fmt.Errorf("invalid development token format")
		}
		
		return &CustomerContext{
			CustomerID:  customerID,
			TenantID:    customerID, // Use customer ID as tenant ID for simplicity
			UserID:      "dev_user",
			WorkspaceID: "dev_workspace",
		}, nil
	}

	// Production token validation would go here
	// Example: JWT validation, API key lookup, etc.
	return nil, fmt.Errorf("invalid token format - use 'dev_customer_<customer_id>' for development")
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
		slog.Default().Warn("failed to create baggage",
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