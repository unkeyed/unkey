package imageref

import (
	_ "crypto/sha256"
	_ "crypto/sha512"
	"fmt"
	"strings"

	referencepkg "github.com/distribution/reference"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

// Public API contracts cap image references at 256 characters. The parser puts
// no bound on total length, so a syntactically valid reference can exceed that
// contract.
const imageLengthMax = 256

// Parse validates and normalizes an image reference. References must include
// an explicit tag or digest so app configuration never relies on latest.
func Parse(raw string) (name.Reference, error) {
	return parse(raw, false)
}

// Normalize validates a reference and returns its canonical name.
func Normalize(raw string) (string, error) {
	parsed, err := Parse(raw)
	if err != nil {
		return "", err
	}
	return parsed.Name(), nil
}

// NormalizeHistorical canonicalizes an image stored before explicit tags were
// required. It makes an implicit latest tag explicit in the returned name.
func NormalizeHistorical(raw string) (string, error) {
	parsed, err := parse(raw, true)
	if err != nil {
		return "", err
	}
	return parsed.Name(), nil
}

// IsDigest reports whether a parsed reference is already immutable.
func IsDigest(reference name.Reference) bool {
	_, ok := reference.(name.Digest)
	return ok
}

// Validate rejects a reference the OCI resolver cannot parse. It preserves the
// legacy API's implicit-latest behavior while new app sources use [Parse].
//
// It does not check that the host serves images at all: github.com/acme/api is
// well-formed and fails at pull time.
func Validate(image string) error {
	if image == "" {
		return fault.New(
			"OCI image reference is empty",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("imageref rejected reference: empty"),
			fault.Public("The OCI image reference is required."),
		)
	}

	if len(image) > imageLengthMax {
		return fault.New(
			"OCI image reference is too long",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal(fmt.Sprintf("imageref rejected reference: %d characters", len(image))),
			fault.Public(fmt.Sprintf("The OCI image reference must not be more than %d characters.", imageLengthMax)),
		)
	}

	if _, err := parse(image, true); err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal(fmt.Sprintf("imageref rejected reference: %q", image)),
			fault.Public(fmt.Sprintf("The OCI image reference %q is not valid. Expected [registry/]repository[:tag][@digest], for example ghcr.io/acme/api:v1.2.3.", image)),
		)
	}

	return nil
}

func parse(raw string, allowImplicitTag bool) (name.Reference, error) {
	reference := strings.TrimSpace(raw)
	if reference == "" {
		return nil, fmt.Errorf("image reference is required")
	}

	lastSlash := strings.LastIndex(reference, "/")
	lastColon := strings.LastIndex(reference, ":")
	if !allowImplicitTag && !strings.Contains(reference, "@") && lastColon <= lastSlash {
		return nil, fmt.Errorf("image reference %q must include an explicit tag or digest", reference)
	}

	normalized, err := referencepkg.ParseDockerRef(reference)
	if err != nil {
		return nil, fmt.Errorf("invalid image reference %q: %w", reference, err)
	}
	parsed, err := name.ParseReference(normalized.String(), name.StrictValidation)
	if err != nil {
		return nil, fmt.Errorf("convert normalized image reference %q: %w", normalized.String(), err)
	}
	return parsed, nil
}
