package assert_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/assert"
)

func TestContains(t *testing.T) {

	t.Run("string contains substring", func(t *testing.T) {
		err := assert.Contains("hello world", "world")
		require.NoError(t, err)
	})

	t.Run("string does not contain substring", func(t *testing.T) {
		err := assert.Contains("hello world", "universe")
		require.Error(t, err)
		require.Contains(t, err.Error(), "string does not contain substring")
	})

	t.Run("with custom message", func(t *testing.T) {
		message := "string should contain the substring"
		err := assert.Contains("hello world", "universe", message)
		require.Error(t, err)
		require.Contains(t, err.Error(), message)
	})
}

// FuzzContains tests the Contains function with fuzzing
func FuzzContains(f *testing.F) {
	// Seed with some examples
	f.Add("hello world", "world")
	f.Add("hello world", "universe")
	f.Add("", "")
	f.Add("", "something")
	f.Add("something", "")

	f.Fuzz(func(t *testing.T, s, substr string) {
		err := assert.Contains(s, substr)
		// nolint:nestif
		if s == "" && substr == "" {
			// Special case: empty string contains empty string
			if err != nil {
				t.Errorf("Contains(%q, %q) should return nil error but got: %v", s, substr, err)
			}
		} else if substr == "" {
			// Special case: any string contains empty string
			if err != nil {
				t.Errorf("Contains(%q, %q) should return nil error but got: %v", s, substr, err)
			}
		} else {
			// General case
			containsSubstr := false
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					containsSubstr = true
					break
				}
			}

			if containsSubstr {
				if err != nil {
					t.Errorf("Contains(%q, %q) should return nil error but got: %v", s, substr, err)
				}
			} else {
				if err == nil {
					t.Errorf("Contains(%q, %q) should return error but got nil", s, substr)
				}
			}
		}
	})
}
