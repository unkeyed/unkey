// Package imageref validates Docker image references before a deployment is
// created, so a bad one is refused up front rather than failing a pull after a
// build slot has been spent on it.
//
// Validate returns nil when the reference is well-formed, or a fault carrying
// the validation code and a caller-facing message.
package imageref

import (
	// go-digest verifies a digest only for algorithms whose hash is linked in, so
	// without these a valid sha256 reference fails as an unsupported algorithm.
	_ "crypto/sha256"
	_ "crypto/sha512"
	"fmt"

	"github.com/distribution/reference"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

// deployments.image is a varchar(256). The parser puts no bound on a reference's
// total length, so a valid one can still be too long for the column, and the row
// is written only after the build: without this an oversized reference fails the
// insert with a build already paid for.
const imageLengthMax = 256

// Validate rejects a reference no registry could serve. Parsing goes through the
// same library the pull path uses, so what passes here is what containerd accepts.
//
// It does not check that the host serves images at all: github.com/acme/api is
// well-formed and fails at pull time.
func Validate(image string) error {
	if image == "" {
		return fault.New(
			"docker image reference is empty",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("imageref rejected reference: empty"),
			fault.Public("The docker image reference is required."),
		)
	}

	if len(image) > imageLengthMax {
		return fault.New(
			"docker image reference is too long",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal(fmt.Sprintf("imageref rejected reference: %d characters", len(image))),
			fault.Public(fmt.Sprintf("The docker image reference must not be more than %d characters.", imageLengthMax)),
		)
	}

	if _, err := reference.ParseDockerRef(image); err != nil {
		// The parser answers most malformed input with a sentinel built at init, so
		// its text names neither the input nor the rule that was broken. It is kept
		// internal for debugging and the caller is told the shape expected instead.
		return fault.Wrap(
			err,
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal(fmt.Sprintf("imageref rejected reference: %q", image)),
			fault.Public(fmt.Sprintf("The docker image reference %q is not valid. Expected [registry/]repository[:tag][@digest], for example ghcr.io/acme/api:v1.2.3.", image)),
		)
	}

	return nil
}
