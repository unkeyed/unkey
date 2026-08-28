// Package imageref validates Docker image references before a deployment is
// created, so a bad one is refused up front rather than failing a pull after a
// build slot has been spent on it.
package imageref

import (
	// go-digest verifies a digest only for algorithms whose hash is linked in, so
	// without these a valid sha256 reference fails as an unsupported algorithm.
	_ "crypto/sha256"
	_ "crypto/sha512"
	"fmt"

	"github.com/distribution/reference"
)

// MaxLength is the width of deployments.image. The grammar allows roughly 456
// characters, so this limit is ours rather than Docker's. The row is written only
// after the build, so without it an oversized reference fails the insert with a
// build already paid for.
const MaxLength = 256

// Validate rejects a reference no registry could serve. Parsing goes through the
// same library the pull path uses, so what passes here is what containerd accepts,
// and the message returned is safe to show a caller.
//
// It does not check that the host serves images at all: github.com/acme/api is
// well-formed and fails at pull time.
func Validate(image string) error {
	if image == "" {
		return fmt.Errorf("docker image reference is required")
	}
	if len(image) > MaxLength {
		return fmt.Errorf("docker image reference must not be more than %d characters", MaxLength)
	}
	if _, err := reference.ParseDockerRef(image); err != nil {
		// The parser's message already opens with "invalid reference format", so
		// prefixing it again reads as a stutter. Only the input is added.
		return fmt.Errorf("%w (%q)", err, image)
	}
	return nil
}
