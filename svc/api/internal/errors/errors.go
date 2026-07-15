package errors

import (
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

func MaskInsufficientPermissionsAsNotFound(err error, code codes.URN, public string) error {
	errCode, ok := fault.GetCode(err)
	if !ok || errCode != codes.Auth.Authorization.InsufficientPermissions.URN() {
		return err
	}

	return fault.New("resource not found",
		fault.Code(code),
		fault.Internal("masking insufficient permissions as not found"),
		fault.Public(public),
	)
}
