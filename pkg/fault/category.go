package fault

import (
	"errors"

	"github.com/unkeyed/unkey/pkg/codes"
)

// GetCategory resolves the attribution category of an error: an explicit
// Category() override if one is set, otherwise the category segment of the
// error's Code. It traverses the error chain like GetCode, returning the first
// match. Returns ("", false) if the chain carries neither an override nor a
// parseable code.
func GetCategory(err error) (codes.Category, bool) {
	for err != nil {
		if w, ok := err.(*wrapped); ok {
			if w.category != "" {
				return w.category, true
			}
			if w.code != "" {
				if c, parseErr := codes.ParseURN(w.code); parseErr == nil {
					return c.Category, true
				}
			}
		}
		err = errors.Unwrap(err)
	}

	return "", false
}
