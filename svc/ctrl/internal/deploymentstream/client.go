package deploymentstream

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/unkeyed/unkey/pkg/cdc"
	binlog "vitess.io/vitess/go/vt/proto/binlogdata"
)

// Client turns topology changes into deployment IDs for Ctrl to look up.
// Watches share a CDC connection and can run at the same time.
type Client struct {
	cdc *cdc.Client
}

// regionPattern rejects unsafe region IDs before adding them to SQL.
var regionPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,48}$`)

// New uses the supplied CDC client, which must not be nil.
// The caller must close that client when all watches have stopped.
func New(client *cdc.Client) *Client {
	return &Client{cdc: client}
}

// Watch reports changes to running deployments in one region.
// An empty token first copies the matching rows. When a row stops matching,
// its old value still provides the deployment ID.
// Invalid regions, invalid IDs, and callback errors stop the watch.
// Resume tokens and checkpoints follow [cdc.Client.Watch].
func (c *Client) Watch(ctx context.Context, region string, token []byte, change func(string) error, checkpoint func([]byte) error) error {
	if !regionPattern.MatchString(region) {
		return errors.New("invalid region ID")
	}
	return c.cdc.Watch(ctx, []cdc.Rule{{
		Table: "deployment_topology",
		Query: fmt.Sprintf("select deployment_id from deployment_topology where region_id = '%s' and desired_status = 'running'", region),
	}}, token, func(event *binlog.VEvent) error {
		for _, rowChange := range event.GetRowEvent().GetRowChanges() {
			row := rowChange.After
			if row == nil {
				row = rowChange.Before
			}
			if row == nil || len(row.Lengths) != 1 || row.Lengths[0] <= 0 || row.Lengths[0] != int64(len(row.Values)) {
				return errors.New("invalid deployment ID in VStream row")
			}
			if err := change(string(row.Values)); err != nil {
				return err
			}
		}
		return nil
	}, checkpoint)
}
