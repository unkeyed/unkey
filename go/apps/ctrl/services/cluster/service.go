package cluster

import (
	"fmt"
	"sync"

	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/buffer"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

type client struct {
	clientID         string
	selectors        map[string]string
	sentinelStates   *buffer.Buffer[*ctrlv1.SentinelState]
	deploymentStates *buffer.Buffer[*ctrlv1.DeploymentState]
	done             chan struct{}
}

func newClient(clientID string, selectors map[string]string) *client {
	return &client{
		clientID:  clientID,
		selectors: selectors,
		deploymentStates: buffer.New[*ctrlv1.DeploymentState](buffer.Config{
			Capacity: 1000,
			Drop:     true,
			Name:     fmt.Sprintf("ctrl_watch_events_%s", clientID),
		}),
		sentinelStates: buffer.New[*ctrlv1.SentinelState](buffer.Config{
			Capacity: 1000,
			Drop:     true,
			Name:     fmt.Sprintf("ctrl_watch_events_%s", clientID),
		}),
		done: make(chan struct{}),
	}
}

type Service struct {
	ctrlv1connect.UnimplementedClusterServiceHandler
	db     db.Database
	logger logging.Logger

	// Maps regions to open clients
	clientsMu sync.RWMutex
	// clientID -> stream
	clients map[string]*client

	// static bearer token for authentication
	bearer string
}

type Config struct {
	Database db.Database
	Logger   logging.Logger
	Bearer   string
}

func New(cfg Config) *Service {
	s := &Service{
		UnimplementedClusterServiceHandler: ctrlv1connect.UnimplementedClusterServiceHandler{},
		db:                                 cfg.Database,
		logger:                             cfg.Logger,
		clientsMu:                          sync.RWMutex{},
		clients:                            make(map[string]*client),
		bearer:                             cfg.Bearer,
	}

	return s
}

var _ ctrlv1connect.ClusterServiceHandler = (*Service)(nil)
