package containers

import (
	"context"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/compose"
)

// ComposeUp starts a docker-compose stack and returns the stack object.
func ComposeUp(t *testing.T) compose.ComposeStack {
	t.Helper()
	os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")

	ctx := context.Background()

	file := path.Join(os.Getenv("OLDPWD"), "deployment/docker-compose.yaml")
	t.Logf("using docker-compose file: %s", file)

	c, err := compose.NewDockerComposeWith(

		compose.WithStackFiles(file),
		compose.StackIdentifier(strings.ToLower(t.Name())),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, c.Down(ctx, compose.RemoveOrphans(true)))
	})

	err = c.Up(ctx, compose.Wait(true))
	require.NoError(t, err)
	return c
}
