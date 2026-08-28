package imageref

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
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

	fits := "ghcr.io/acme/" + strings.Repeat("a", MaxLength-len("ghcr.io/acme/"))
	require.Len(t, fits, MaxLength)
	require.NoError(t, Validate(fits))
	require.Error(t, Validate(fits+"a"))
}
