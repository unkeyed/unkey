package network

import (
	"fmt"
	"log/slog"
	"net"
	"sync"
)

// NewManager creates a new network manager to handle bridge/tap creation
func NewManager(logger *slog.Logger, config *Config) (*Manager, error) {
	if config == nil {
		logger.Error("creating network manager")
		return nil, fmt.Errorf("network config can not be nil")
	}

	logger.Info("creating network manager",
		slog.String("bridge_name", config.BridgeName),
		slog.String("base_network", config.BaseNetwork.String()),
	)

	m := &Manager{ //nolint:exhaustruct
		logger: logger,
		config: config,
	}

	return m, nil
}

// Config holds network configuration
type Config struct {
	BaseNetwork     *net.IPNet
	BridgeName      string
	DNSServers      []string // Default: ["8.8.8.8", "8.8.4.4"]
	EnableIPv6      bool
	EnableRateLimit bool
	RateLimitMbps   int // Per VM rate limit in Mbps
}

type Manager struct {
	logger   *slog.Logger
	config   *Config
	mu       sync.RWMutex
	bridgeMu sync.RWMutex
}
