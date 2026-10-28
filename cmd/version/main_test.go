package version_test

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/version"
	"github.com/unkeyed/unkey/pkg/buildinfo"
)

func TestVersionShowsTheInstalledVersion(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "stdout")
	require.NoError(t, err)
	stdout := os.Stdout
	os.Stdout = file
	t.Cleanup(func() {
		os.Stdout = stdout
		require.NoError(t, file.Close())
	})
	require.NoError(t, version.Cmd.Run(context.Background(), []string{"unkey version"}))
	_, err = file.Seek(0, io.SeekStart)
	require.NoError(t, err)
	out, err := io.ReadAll(file)
	require.NoError(t, err)
	require.Equal(t, buildinfo.Version+"\n", string(out))
}

func TestVersionDoesNotPretendToManageDeployments(t *testing.T) {
	for _, args := range [][]string{{"get", "example"}, {"list"}, {"rollback", "example.com", "example"}} {
		err := version.Cmd.Run(context.Background(), append([]string{"unkey version"}, args...))
		require.Error(t, err)
	}
}
