package errors

import (
	"fmt"
	"math"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
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

// MaskInsufficientPermissionsAsNotFound replaces an insufficient-permissions
// error with a not-found error so callers cannot use response differences to
// find resources. It returns all other errors unchanged.
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
