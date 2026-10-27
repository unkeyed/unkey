package notificationapi_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/notificationapi"
)

func TestIdempotencyKeysPreserveTheirExactUTF8Identity(t *testing.T) {
	for _, tc := range []struct {
		name  string
		key   string
		valid bool
	}{
		{name: "single character", key: "a", valid: true},
		{name: "ascii byte limit", key: strings.Repeat("a", 128), valid: true},
		{name: "utf8 byte limit", key: strings.Repeat("é", 64), valid: true},
		{name: "internal whitespace", key: "a b", valid: true},
		{name: "empty"},
		{name: "ascii over limit", key: strings.Repeat("a", 129)},
		{name: "utf8 over limit", key: strings.Repeat("é", 64) + "a"},
		{name: "leading whitespace", key: " key"},
		{name: "trailing whitespace", key: "key\n"},
		{name: "leading unicode whitespace", key: "\u00a0key"},
		{name: "trailing unicode whitespace", key: "key\u2003"},
		{name: "nul", key: "key\x00suffix"},
		{name: "invalid utf8", key: string([]byte{0xff})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := notificationapi.ValidateIdempotencyKey(tc.key)
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
