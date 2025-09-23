package network

import (
	"fmt"
	"log/slog"

	"github.com/vishvananda/netlink"
)

// CreateBridge creates the bridge if it doesn't exist
func CreateBridge(logger *slog.Logger, cfg Config) (string, error) {
	link, linkErr := netlink.LinkByName(cfg.BridgeName)
	if linkErr == nil {
		logger.Debug("bridge exists",
			slog.String("bridge", cfg.BridgeName),
			slog.Int("mtu", link.Attrs().MTU),
		)
		return link.Attrs().Name, nil
	}

	bridge := &netlink.Bridge{ //nolint:exhaustruct
		LinkAttrs: netlink.LinkAttrs{
			Name: cfg.BridgeName, //nolint:exhaustruct
		},
	}

	if err := netlink.LinkAdd(bridge); err != nil {
		logger.Error("failed to create bridge",
			slog.String("bridge", cfg.BridgeName),
			slog.String("error", err.Error()),
		)
		return "", fmt.Errorf("failed to create bridge: %w", err)
	}

	br, brErr := netlink.LinkByName(cfg.BridgeName)
	if brErr != nil {
		return "", fmt.Errorf("failed to get bridge: %w", brErr)
	}

	addr, err := netlink.ParseAddr(cfg.BaseNetwork.String())
	if err != nil {
		return "", fmt.Errorf("failed to parse bridge IP: %w", err)
	}

	if err := netlink.AddrAdd(br, addr); err != nil {
		logger.Error("failed to add IP to bridge",
			slog.String("bridge", cfg.BridgeName),
			slog.String("error", err.Error()),
		)
		return "", fmt.Errorf("failed to add IP to bridge: %w", err)
	}

	// Bring bridge up
	if err := netlink.LinkSetUp(br); err != nil {
		logger.Error("failed to bring bridge up",
			slog.String("bridge", cfg.BridgeName),
			slog.String("error", err.Error()),
		)
		return "", fmt.Errorf("failed to bring bridge up: %w", err)
	}

	logger.Debug("bridge created",
		slog.String("bridge", cfg.BridgeName),
		slog.Int("mtu", br.Attrs().MTU),
	)

	return cfg.BridgeName, nil
}
