package deploymentstream

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/unkeyed/unkey/pkg/cdc"
)

// Client turns topology changes into deployment IDs for Ctrl to look up.
// Each watch owns its CDC watcher. Watches can run at the same time.
type Client struct {
	config cdc.Config
}

// regionPattern rejects unsafe region IDs before adding them to SQL.
var regionPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,48}$`)

// New checks endpoint settings without opening a connection. It returns nil on error.
// Watch supplies deployment rules instead of using cfg.Rules.
func New(cfg cdc.Config) (*Client, error) {
	if err := cfg.ValidateEndpoint(); err != nil {
		return nil, err
	}
	cfg.Rules = nil
	return &Client{config: cfg}, nil
}

// Watch reports changes to running deployments in one region.
// An empty token first copies the matching rows. When a row stops matching,
// its old value still provides the deployment ID.
// Invalid regions, invalid IDs, and callback errors stop the watch.
// The CDC watcher closes when the watch ends. Tokens follow [cdc.Watcher.Watch].
func (c *Client) Watch(ctx context.Context, region string, token []byte, apply func(Event) error) (err error) {
	if !regionPattern.MatchString(region) {
		return errors.New("invalid region ID")
	}
	cfg := c.config
	cfg.Rules = []cdc.Rule{{
		Table: "deployment_topology",
		Query: fmt.Sprintf("select deployment_id from deployment_topology where region_id = '%s' and desired_status = 'running'", region),
	}}
	watcher, err := cdc.New(cfg)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, watcher.Close()) }()
	return watcher.Watch(ctx, token, func(event cdc.Event) error {
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
