package privatenetwork

import (
	"context"
	"errors"
	"sync"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/cdc"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// Config requires a database and VStream for the same source, and a clock.
type Config struct {
	Database db.Database
	VStream  cdc.Config
	Clock    clock.Clock
}

// Catalogs keeps one catalog per platform. A platform's catalog starts on its
// first [Catalogs.Snapshot] and stops when [Catalogs.Run] returns.
type Catalogs struct {
	store   store
	vstream cdc.Config
	clock   clock.Clock

	mu       sync.Mutex
	ctx      context.Context
	stopped  bool
	running  sync.WaitGroup
	catalogs map[string]*catalog
}

// New checks the configuration without opening a stream.
func New(cfg Config) (*Catalogs, error) {
	if err := assert.All(
		assert.NotNil(cfg.Database, "private network catalogs require a database"),
		assert.NotNil(cfg.Clock, "private network catalogs require a clock"),
		cfg.VStream.ValidateEndpoint(),
	); err != nil {
		return nil, err
	}
	return &Catalogs{
		store:    dbStore{database: cfg.Database},
		vstream:  cfg.VStream,
		clock:    cfg.Clock,
		mu:       sync.Mutex{},
		ctx:      nil,
		stopped:  false,
		running:  sync.WaitGroup{},
		catalogs: map[string]*catalog{},
	}, nil
}

// Run keeps started catalogs current until ctx ends, then waits for them.
func (c *Catalogs) Run(ctx context.Context) error {
	c.mu.Lock()
	if c.ctx != nil {
		c.mu.Unlock()
		return errors.New("private network catalogs are already running")
	}
	c.ctx = ctx
	c.mu.Unlock()

	<-ctx.Done()
	c.mu.Lock()
	c.stopped = true
	c.mu.Unlock()
	c.running.Wait()
	return nil
}

// Snapshot returns the platform's catalog. Before the first catalog is built it
// waits until ctx ends. Afterwards it returns [ErrUnavailable] at once while
// the catalog's CDC stream has been silent for 15 seconds.
func (c *Catalogs) Snapshot(ctx context.Context, platform string) (Snapshot, error) {
	platformCatalog, err := c.catalog(platform)
	if err != nil {
		return Snapshot{Connections: nil, Topology: nil, Version: "", ID: "", Certified: false}, err
	}
	return platformCatalog.served.wait(ctx)
}

func (c *Catalogs) catalog(platform string) (*catalog, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if existing, ok := c.catalogs[platform]; ok {
		return existing, nil
	}
	if c.ctx == nil || c.stopped {
		return nil, ErrUnavailable
	}
	cfg := c.vstream
	cfg.Rules = cdcRules()
	watcher, err := cdc.New(cfg)
	if err != nil {
		return nil, err
	}
	platformCatalog := newCatalog(platform, c.store, watcher.Watch, c.clock)
	c.catalogs[platform] = platformCatalog
	c.running.Go(func() {
		platformCatalog.run(c.ctx)
		if err := watcher.Close(); err != nil {
			logger.Warn("close private network catalog stream", "platform", platform, "error", err)
		}
	})
	return platformCatalog, nil
}
