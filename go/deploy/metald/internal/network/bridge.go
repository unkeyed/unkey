package network

import (
	"fmt"
	"log/slog"

	"github.com/vishvananda/netlink"
)

// ensureBridge creates the bridge if it doesn't exist
func (m *Manager) ensureBridge() error {
	if link, err := netlink.LinkByName(m.config.BridgeName); err == nil {
		m.logger.Debug("bridge exists",
			slog.String("bridge", m.config.BridgeName),
			slog.String("type", link.Type()),
			slog.String("state", link.Attrs().OperState.String()),
		)
		return nil
	}

	bridge := &netlink.Bridge{ //nolint:exhaustruct
		LinkAttrs: netlink.LinkAttrs{
			Name: m.config.BridgeName, //nolint:exhaustruct
		},
	}

	if err := netlink.LinkAdd(bridge); err != nil {
		m.logger.Error("failed to create bridge",
			slog.String("bridge", m.config.BridgeName),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to create bridge: %w", err)
	}

	br, err := netlink.LinkByName(m.config.BridgeName)
	if err != nil {
		return fmt.Errorf("failed to get bridge: %w", err)
	}

	addr, err := netlink.ParseAddr(m.config.BaseNetwork.String())
	if err != nil {
		return fmt.Errorf("failed to parse bridge IP: %w", err)
	}

	if err := netlink.AddrAdd(br, addr); err != nil {
		m.logger.Error("failed to add IP to bridge",
			slog.String("bridge", m.config.BridgeName),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to add IP to bridge: %w", err)
	}

	// Bring bridge up
	if err := netlink.LinkSetUp(br); err != nil {
		m.logger.Error("failed to bring bridge up",
			slog.String("bridge", m.config.BridgeName),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to bring bridge up: %w", err)
	}

	return nil
}
