package imageref

import (
	"strings"
	"testing"

	"github.com/distribution/reference"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

func TestValidate(t *testing.T) {
	t.Parallel()

	digest := "sha256:" + strings.Repeat("a", 64)

	valid := []string{
		"kebap",
		"nginx:1.27",
		"library/redis:7",
		"ghcr.io/acme/api:v1.2.3",
		"docker.io/library/mysql:8.0.36",
		"localhost:5000/acme/api:dev",
		"registry.internal:5000/team/api",
		"ghcr.io/acme/api@" + digest,
		"ghcr.io/acme/api:v1@" + digest,
		// A tag is case sensitive, so uppercase in one is legal.
		"ghcr.io/acme/api:V1.2-RC1",
	}
	for _, image := range valid {
		require.NoError(t, Validate(image), image)
	}

	invalid := []string{
		"",
		"Acme/Api:v1",
		"ghcr.io/Acme/api",
		"my image:v1",
		"https://ghcr.io/acme/api",
		"ghcr.io/acme/api:",
		"ghcr.io/acme/api@sha256:abc123",
		"ghcr.io//acme/api",
		// One character over what deployments.image holds.
		"ghcr.io/acme/" + strings.Repeat("a", 244),
	}
	for _, image := range invalid {
		require.Error(t, Validate(image), image)
	}
}

// TestValidateLengthBoundary pins the cap to the column width rather
// than to whatever the grammar allows.
func TestValidateLengthBoundary(t *testing.T) {
	t.Parallel()

	fits := "ghcr.io/acme/" + strings.Repeat("a", imageLengthMax-len("ghcr.io/acme/"))
	require.Len(t, fits, imageLengthMax)
	require.NoError(t, Validate(fits))
	require.Error(t, Validate(fits+"a"))
}

// TestValidateMessages pins the caller-facing text, which handlers render
// straight into an API response, and keeps the parser's own wording internal.
func TestValidateMessages(t *testing.T) {
	t.Parallel()

	err := Validate("ghcr.io/acme/api:v1 KEBAP")
	require.Equal(t,
		`The OCI image reference "ghcr.io/acme/api:v1 KEBAP" is not valid. Expected [registry/]repository[:tag][@digest], for example ghcr.io/acme/api:v1.2.3.`,
		fault.UserFacingMessage(err))
	require.NotContains(t, fault.UserFacingMessage(err), "could not parse reference")
	require.ErrorIs(t, err, reference.ErrReferenceInvalidFormat)

	code, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.App.Validation.InvalidInput.URN(), code)

	require.Equal(t, "The OCI image reference is required.", fault.UserFacingMessage(Validate("")))
	require.Equal(t,
		"The OCI image reference must not be more than 256 characters.",
		fault.UserFacingMessage(Validate(strings.Repeat("a", imageLengthMax+1))))
}

// TestValidateAlwaysFaults keeps every rejection on the fault path, so a caller
// rendering UserFacingMessage never falls back to an empty string.
func TestValidateAlwaysFaults(t *testing.T) {
	t.Parallel()

	for _, image := range []string{"", "Acme/Api:v1", "my image:v1", strings.Repeat("a", imageLengthMax+1)} {
		err := Validate(image)
		require.Error(t, err, image)
		require.NotEmpty(t, fault.UserFacingMessage(err), image)
	}
}
