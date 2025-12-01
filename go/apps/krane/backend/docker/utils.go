package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
)

// deleteByLabels removes all containers matching the specified label filters.
//
// This is a utility function that implements idempotent container deletion.
// It finds containers by label filters and forcibly removes them with volumes
// and network links to ensure complete cleanup.
//
// The function is idempotent - it will not fail if containers don't exist or
// are already removed.
func (d *docker) deleteByLabels(ctx context.Context, labels map[string]string) error {

	labelFilters := filters.NewArgs()
	for k, v := range labels {
		labelFilters.Add(k, v)
	}
	containers, err := d.client.ContainerList(ctx, container.ListOptions{
		Size:    false,
		Latest:  false,
		Since:   "",
		Before:  "",
		Limit:   0,
		All:     true,
		Filters: labelFilters,
	})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	for _, c := range containers {
		err := d.client.ContainerRemove(ctx, c.ID, container.RemoveOptions{
			RemoveVolumes: true,
			RemoveLinks:   true,
			Force:         true,
		})
		if err != nil {
			// Check if container doesn't exist (idempotent)
			if strings.Contains(err.Error(), "No such container") {
				d.logger.Info("container already removed, skipping",
					"container_id", c.ID,
				)
				continue
			}
			return fmt.Errorf("failed to remove container %s: %w", c.ID, err)
		}
		d.logger.Info("container removed successfully",
			"container_id", c.ID,
		)
	}

	return nil
}
