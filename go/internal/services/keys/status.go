package keys

import (
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

// KeyStatus represents the validation status of a key after verification.
// Each status indicates a specific validation outcome that can be used
// to determine the appropriate response and error handling.
type KeyStatus string

const (
	StatusValid                   KeyStatus = "VALID"
	StatusNotFound                KeyStatus = "NOT_FOUND"
	StatusDisabled                KeyStatus = "DISABLED"
	StatusExpired                 KeyStatus = "EXPIRED"
	StatusForbidden               KeyStatus = "FORBIDDEN"
	StatusInsufficientPermissions KeyStatus = "INSUFFICIENT_PERMISSIONS"
	StatusRateLimited             KeyStatus = "RATE_LIMITED"
	StatusUsageExceeded           KeyStatus = "USAGE_EXCEEDED"
	StatusWorkspaceDisabled       KeyStatus = "WORKSPACE_DISABLED"
	StatusWorkspaceNotFound       KeyStatus = "WORKSPACE_NOT_FOUND"
)

// ToFault converts the verification result to an appropriate fault error.
// This method should only be called when k.Valid is false.
// It provides structured error information that matches the API specification.
func (k *KeyVerifier) ToFault() error {
	switch k.Status {
	case StatusValid:
		return nil
	case StatusNotFound:
		return fault.New("key does not exist",
			fault.Code(codes.Auth.Authentication.KeyNotFound.URN()),
			fault.Internal("key does not exist"),
			fault.Public("We could not find the requested key."),
		)
	case StatusDisabled:
		message := k.message
		if message == "" {
			message = "the key is disabled"
		}
		return fault.New("key is disabled",
			fault.Code(codes.Auth.Authorization.KeyDisabled.URN()),
			fault.Internal(message),
			fault.Public("The key is disabled."),
		)
	case StatusExpired:
		message := k.message
		if message == "" {
			message = "the key has expired"
		}
		return fault.New("key has expired",
			fault.Code(codes.Auth.Authorization.Forbidden.URN()),
			fault.Internal(message),
			fault.Public(message),
		)
	case StatusWorkspaceDisabled:
		return fault.New("workspace is disabled",
			fault.Code(codes.Auth.Authorization.WorkspaceDisabled.URN()),
			fault.Internal("workspace disabled"),
			fault.Public("The workspace is disabled."),
		)
	case StatusWorkspaceNotFound:
		return fault.New("workspace not found",
			fault.Code(codes.Data.Workspace.NotFound.URN()),
			fault.Internal("workspace disabled"),
			fault.Public("The requested workspace does not exist."),
		)
	case StatusForbidden:
		message := k.message
		if message == "" {
			message = "Forbidden"
		}
		return fault.New("forbidden",
			fault.Code(codes.Auth.Authorization.Forbidden.URN()),
			fault.Internal(message),
			fault.Public(message),
		)
	case StatusInsufficientPermissions:
		message := k.message
		if message == "" {
			message = "Insufficient permissions to access this resource."
		}

		return fault.New("insufficient permissions",
			fault.Code(codes.Auth.Authorization.InsufficientPermissions.URN()),
			fault.Internal(message),
			fault.Public(message),
		)
	case StatusUsageExceeded:
		message := k.message
		if message == "" {
			message = "Key usage limit exceeded."
		}
		return fault.New("key usage limit exceeded",
			fault.Code(codes.Auth.Authorization.Forbidden.URN()),
			fault.Internal(message),
			fault.Public(message),
		)
	case StatusRateLimited:
		message := k.message
		if message == "" {
			message = "Rate limit exceeded"
		}
		return fault.New("rate limit exceeded",
			fault.Code(codes.Auth.Authorization.Forbidden.URN()),
			fault.Internal(message),
			fault.Public(message),
		)
	default:
		return fault.New("key verification failed",
			fault.Code(codes.Auth.Authorization.Forbidden.URN()),
			fault.Internal("key verification failed with unknown status"),
			fault.Public("Key verification failed."),
		)
	}
}

// ToOpenAPIStatus converts our internal KeyStatus to the OpenAPI response status type.
// This mapping ensures consistency between internal validation and external API responses.
func (k *KeyVerifier) ToOpenAPIStatus() openapi.V2KeysVerifyKeyResponseDataCode {
	switch k.Status {
	case StatusValid:
		return openapi.VALID
	case StatusNotFound:
		return openapi.NOTFOUND
	case StatusDisabled:
		return openapi.DISABLED
	case StatusExpired:
		return openapi.EXPIRED
	case StatusForbidden:
		return openapi.FORBIDDEN
	case StatusInsufficientPermissions:
		return openapi.INSUFFICIENTPERMISSIONS
	case StatusUsageExceeded:
		return openapi.USAGEEXCEEDED
	case StatusRateLimited:
		return openapi.RATELIMITED
	case StatusWorkspaceNotFound:
		return openapi.NOTFOUND
	case StatusWorkspaceDisabled:
		return openapi.FORBIDDEN
	default:
		return openapi.FORBIDDEN
	}
}

// setInvalid marks the key as invalid with the specified status and message.
// This is used internally by validation methods to indicate validation failures.
func (k *KeyVerifier) setInvalid(status KeyStatus, message string) {
	k.Status = status
	k.message = message
}
