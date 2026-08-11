// Package projectgate holds project rules shared by the public API and ctrl.
package projectgate

import (
	"fmt"
	"strings"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

// DefaultSlug identifies the workspace's internal ownership project.
const DefaultSlug = "default"

// CheckSlug rejects values that collide with the internal default project
// under MySQL's case-insensitive project-slug collation.
func CheckSlug(slug string) error {
	if !strings.EqualFold(slug, DefaultSlug) {
		return nil
	}

	return fault.New(
		"project slug is reserved",
		fault.Code(codes.App.Validation.InvalidInput.URN()),
		fault.Internal("default project slug is reserved"),
		fault.Public(fmt.Sprintf("The project slug '%s' is reserved.", slug)),
	)
}
