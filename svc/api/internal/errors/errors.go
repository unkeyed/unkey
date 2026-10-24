package errors

import (
	"fmt"
	"math"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
)

// humanizeBytes formats a byte count using decimal (SI) units, e.g. 16000 ->
// "16 kB" and 1048576 -> "1.0 MB". Values below 10 bytes render as "N B".
// Larger values use one decimal place when the mantissa is below 10 and none
// otherwise, so "1.5 MB" but "16 kB".
func humanizeBytes(s uint64) string {
	sizes := []string{"B", "kB", "MB", "GB", "TB", "PB", "EB"}
	if s < 10 {
		return fmt.Sprintf("%d B", s)
	}
	const base = 1000.0
	e := math.Floor(math.Log(float64(s)) / math.Log(base))
	val := math.Floor(float64(s)/math.Pow(base, e)*10+0.5) / 10
	format := "%.0f %s"
	if val < 10 {
		format = "%.1f %s"
	}
	return fmt.Sprintf(format, val, sizes[int(e)])
}

// MaxByteSize returns a 400 validation error when sizeBytes exceeds limitBytes,
// reporting both sizes human-readably (e.g. "1.0 MB", "16 kB"), and nil when
// within the limit. subject is the sentence subject placed before "is too
// large", so pass text that reads naturally there, e.g. "Metadata" or
// fmt.Sprintf("Variable %q value", key).
//
// The budget is UTF-8 bytes, not code points: OpenAPI maxLength counts code
// points, so a multibyte value can pass schema validation and still overflow a
// byte-bounded store. Callers pass len(value) or len(marshaled), which are
// already byte counts.
func MaxByteSize(subject string, sizeBytes, limitBytes int) error {
	if sizeBytes <= limitBytes {
		return nil
	}
	return fault.New("value exceeds byte limit",
		fault.Code(codes.App.Validation.InvalidInput.URN()),
		fault.Internal("value exceeds byte limit"),
		fault.Public(fmt.Sprintf(
			"%s is too large: it must be at most %s, but is %s.",
			subject,
			humanizeBytes(uint64(limitBytes)),
			humanizeBytes(uint64(sizeBytes)),
		)),
	)
}

// MaskInsufficientPermissionsAsNotFound hides a resource from a caller who
// cannot read it, and tells a caller who can read it which permission is missing.
//
// granted and query are the principal permissions and the query that produced
// err. When err is an insufficient-permissions denial and the caller cannot
// read the resource, the result is a fresh not-found fault using code and
// public, with no permission text attached. When the caller can read the
// resource, the result is a fresh insufficient-permissions fault whose public
// message names the missing permission or permissions and says to log in again
// or grant them. Any other error, including a nil error, is returned unchanged.
//
// The result is never a wrap of err. Public messages in a fault chain are
// concatenated into the response, and the denial names resource ids.
func MaskInsufficientPermissionsAsNotFound(err error, granted []string, query rbac.PermissionQuery, code codes.URN, public string) error {
	return MaskInsufficientPermissionsWithVisibility(err, granted, query, query, code, public)
}

// MaskInsufficientPermissionsWithVisibility is
// [MaskInsufficientPermissionsAsNotFound] for a route whose not-found response
// hides a different resource than the one required names, such as a create
// route that hides the parent it creates under. visibility is the read query
// for the resource the not-found response hides. required is the query that
// produced err, so the 403 names the action the route demanded rather than
// offering the read as a substitute.
func MaskInsufficientPermissionsWithVisibility(err error, granted []string, required, visibility rbac.PermissionQuery, code codes.URN, public string) error {
	errCode, ok := fault.GetCode(err)
	if !ok || errCode != codes.Auth.Authorization.InsufficientPermissions.URN() {
		return err
	}
	if rbac.CanRead(visibility, granted) {
		return readablePermissionDenied(required, granted)
	}
	return fault.New("resource not found",
		fault.Code(code),
		fault.Internal("masking insufficient permissions as not found"),
		fault.Public(public),
	)
}

func readablePermissionDenied(required rbac.PermissionQuery, granted []string) error {
	return fault.New("insufficient permissions",
		fault.Code(codes.Auth.Authorization.InsufficientPermissions.URN()),
		fault.Internal("caller can read the resource but lacks a required permission"),
		fault.Public(rbac.PermissionDeniedDetail(required, granted)),
	)
}
