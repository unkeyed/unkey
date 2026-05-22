package match

import (
	"errors"
	"testing"

	"github.com/unkeyed/unkey/pkg/fuzz"
)

func FuzzMatchWatchPaths(f *testing.F) {
	fuzz.Seed(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)
		patterns := fuzz.Slice[string](c)
		files := fuzz.Slice[string](c)

		for _, file := range files {
			for _, pattern := range patterns {
				_, err := matchWatchPath(pattern, file)
				if err != nil && !errors.Is(err, errInvalidWatchPath) {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		}
	})
}
