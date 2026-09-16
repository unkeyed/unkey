package deploymentstream

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/unkeyed/unkey/pkg/cdc"
)

// Client turns topology changes into deployment IDs for Ctrl to look up.
// Watches share a CDC connection and can run at the same time.
type Client struct {
	connection *cdc.Connection
}

// Event contains either a deployment ID to look up or a checkpoint token.
// Watch sets exactly one of these fields.
type Event struct {
	DeploymentID string
	ResumeToken  []byte
}

// regionPattern rejects unsafe region IDs before adding them to SQL.
var regionPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,48}$`)

// New uses the supplied CDC connection, which must not be nil.
// The caller must close that connection when all watches have stopped.
func New(connection *cdc.Connection) *Client {
	return &Client{connection: connection}
}

// Watch reports changes to running deployments in one region.
// An empty token first copies the matching rows. When a row stops matching,
// its old value still provides the deployment ID.
// Invalid regions, invalid IDs, and callback errors stop the watch.
// Resume tokens and checkpoints follow [cdc.Client.Forward].
func (c *Client) Watch(ctx context.Context, region string, token []byte, apply func(Event) error) error {
	if !regionPattern.MatchString(region) {
		return errors.New("invalid region ID")
	}
	client, err := cdc.New(cdc.Config{
		Connection: c.connection,
		Rules: []cdc.Rule{{
			Table: "deployment_topology",
			Query: fmt.Sprintf("select deployment_id from deployment_topology where region_id = '%s' and desired_status = 'running'", region),
		}},
		ResumeToken: token,
	})
	if err != nil {
		return err
	}
	return client.Forward(ctx, func(event cdc.Event) error {
		if event.Change == nil {
			return apply(Event{DeploymentID: "", ResumeToken: event.ResumeToken})
		}
		for _, rowChange := range event.Change.GetRowEvent().GetRowChanges() {
			row := rowChange.After
			if row == nil {
				row = rowChange.Before
			}
			if row == nil || len(row.Lengths) != 1 || row.Lengths[0] <= 0 || row.Lengths[0] != int64(len(row.Values)) {
				return errors.New("invalid deployment ID in VStream row")
			}
			if err := apply(Event{DeploymentID: string(row.Values), ResumeToken: nil}); err != nil {
				return err
			}
		}
		return nil
	})
}
