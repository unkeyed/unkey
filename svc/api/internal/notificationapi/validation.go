package notificationapi

import (
	"strings"
	"unicode/utf8"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

func ValidateIdempotencyKey(value string) error {
	if value == "" || len(value) > 128 || value != strings.TrimSpace(value) || strings.ContainsRune(value, '\x00') || !utf8.ValidString(value) {
		return fault.New("invalid notification idempotency key",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Provide an idempotency key containing 1 to 128 UTF-8 bytes, without leading or trailing whitespace or NUL characters."),
		)
	}
	return nil
}
