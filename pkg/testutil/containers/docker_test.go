package containers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/errdefs"
	"github.com/stretchr/testify/require"
)

func TestCreateOrAttachReusableContainer_AttachesAfterConflict(t *testing.T) {
	t.Parallel()

	existing := &Container{
		ID:    "existing",
		Host:  "localhost",
		Ports: map[string]string{},
	}
	createCalls := 0
	inspectCalls := 0

	resp, ctr, created, err := createOrAttachReusableContainer(
		context.Background(),
		time.Second,
		0,
		func() (container.CreateResponse, error) {
			createCalls++
			return container.CreateResponse{}, errdefs.Conflict(errors.New("container name already exists"))
		},
		func() (*Container, bool, error) {
			inspectCalls++
			return existing, true, nil
		},
	)

	require.NoError(t, err)
	require.False(t, created)
	require.Empty(t, resp.ID)
	require.Same(t, existing, ctr)
	require.Equal(t, 1, createCalls)
	require.Equal(t, 1, inspectCalls)
}

func TestCreateOrAttachReusableContainer_RetriesCreateWhenConflictIsNotInspectable(t *testing.T) {
	t.Parallel()

	createCalls := 0
	inspectCalls := 0

	resp, ctr, created, err := createOrAttachReusableContainer(
		context.Background(),
		time.Second,
		0,
		func() (container.CreateResponse, error) {
			createCalls++
			if createCalls == 1 {
				return container.CreateResponse{}, errdefs.Conflict(errors.New("container name already exists"))
			}
			return container.CreateResponse{ID: "created"}, nil
		},
		func() (*Container, bool, error) {
			inspectCalls++
			return nil, false, nil
		},
	)

	require.NoError(t, err)
	require.True(t, created)
	require.Equal(t, "created", resp.ID)
	require.Nil(t, ctr)
	require.Equal(t, 2, createCalls)
	require.Equal(t, 1, inspectCalls)
}
