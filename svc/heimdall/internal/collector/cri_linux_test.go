//go:build linux

package collector

import (
	"testing"

	eventstypes "github.com/containerd/containerd/api/events"
	"github.com/containerd/typeurl/v2"
	"github.com/stretchr/testify/require"
)

func TestCRIWatcherIgnoresExecExitBeforeContainerLookup(t *testing.T) {
	event, err := typeurl.MarshalAny(&eventstypes.TaskExit{
		ContainerID: "container-id",
		ID:          "exec-id",
	})
	require.NoError(t, err)

	watcher := &criWatcher{}
	require.NotPanics(t, func() {
		watcher.handle(t.Context(), "/tasks/exit", event)
	})
}
