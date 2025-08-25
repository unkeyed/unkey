package network

import (
	"fmt"
	"log/slog"
	"net"
	"os/exec"
	"strings"
)

// These functions manage port forwarding rules using nftables instead of iptables

// setupPortForwarding creates nftables DNAT rule for port forwarding
func (m *Manager) setupPortForwarding(vmIP net.IP, mapping PortMapping) error {
	// Create nftables DNAT rule: host:hostPort -> vmIP:containerPort
	rule := fmt.Sprintf(
		"nft add rule ip nat PREROUTING tcp dport %d dnat to %s:%d",
		mapping.HostPort, vmIP.String(), mapping.ContainerPort,
	)

	if mapping.Protocol == "udp" {
		rule = fmt.Sprintf(
			"nft add rule ip nat PREROUTING udp dport %d dnat to %s:%d",
			mapping.HostPort, vmIP.String(), mapping.ContainerPort,
		)
	}

	m.logger.Info("creating port forwarding rule",
		slog.String("vm_id", mapping.VMID),
		slog.String("vm_ip", vmIP.String()),
		slog.Int("host_port", mapping.HostPort),
		slog.Int("container_port", mapping.ContainerPort),
		slog.String("protocol", mapping.Protocol),
		slog.String("rule", rule),
	)

	cmd := exec.Command("bash", "-c", rule)
	if output, err := cmd.CombinedOutput(); err != nil {
		m.logger.Error("failed to create port forwarding rule",
			slog.String("rule", rule),
			slog.String("error", err.Error()),
			slog.String("output", string(output)),
		)
		return fmt.Errorf("failed to create port forwarding rule: %w", err)
	}

	m.logger.Info("port forwarding rule created successfully",
		slog.String("vm_id", mapping.VMID),
		slog.Int("host_port", mapping.HostPort),
		slog.Int("container_port", mapping.ContainerPort),
		slog.String("protocol", mapping.Protocol),
	)

	return nil
}

// removePortForwarding removes nftables DNAT rule for port forwarding
func (m *Manager) removePortForwarding(vmIP net.IP, mapping PortMapping) error {
	// List rules to find the handle, then delete by handle
	// This is more reliable than trying to match the exact rule text

	listCmd := fmt.Sprintf("nft --handle list chain ip nat PREROUTING")
	cmd := exec.Command("bash", "-c", listCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		m.logger.Warn("failed to list nftables rules for cleanup",
			slog.String("error", err.Error()),
			slog.String("output", string(output)),
		)
		return nil // Non-fatal - rule might not exist
	}

	// Look for rule containing our DNAT target
	target := fmt.Sprintf("dnat to %s:%d", vmIP.String(), mapping.ContainerPort)
	portMatch := fmt.Sprintf("dport %d", mapping.HostPort)

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, target) && strings.Contains(line, portMatch) && strings.Contains(line, mapping.Protocol) {
			// Extract handle from line like "tcp dport 8080 dnat to 172.16.1.10:9999 # handle 5"
			if handlePos := strings.Index(line, "# handle "); handlePos != -1 {
				handleStr := strings.TrimSpace(line[handlePos+9:])
				deleteCmd := fmt.Sprintf("nft delete rule ip nat PREROUTING handle %s", handleStr)

				m.logger.Info("removing port forwarding rule",
					slog.String("vm_id", mapping.VMID),
					slog.String("vm_ip", vmIP.String()),
					slog.Int("host_port", mapping.HostPort),
					slog.Int("container_port", mapping.ContainerPort),
					slog.String("protocol", mapping.Protocol),
					slog.String("handle", handleStr),
				)

				delCmd := exec.Command("bash", "-c", deleteCmd)
				if delOutput, delErr := delCmd.CombinedOutput(); delErr != nil {
					m.logger.Warn("failed to remove port forwarding rule",
						slog.String("command", deleteCmd),
						slog.String("error", delErr.Error()),
						slog.String("output", string(delOutput)),
					)
				} else {
					m.logger.Info("port forwarding rule removed successfully",
						slog.String("vm_id", mapping.VMID),
						slog.Int("host_port", mapping.HostPort),
						slog.String("handle", handleStr),
					)
				}
				break
			}
		}
	}

	return nil
}

// ensureNftablesTable ensures the required nftables table and chain exist
func (m *Manager) ensureNftablesTable() error {
	// Create nat table if it doesn't exist
	tableCmd := "nft add table ip nat 2>/dev/null || true"
	if err := exec.Command("bash", "-c", tableCmd).Run(); err != nil {
		m.logger.Warn("failed to ensure nat table exists", slog.String("error", err.Error()))
	}

	// Create PREROUTING chain if it doesn't exist
	chainCmd := "nft add chain ip nat PREROUTING '{ type nat hook prerouting priority -100; }' 2>/dev/null || true"
	if err := exec.Command("bash", "-c", chainCmd).Run(); err != nil {
		m.logger.Warn("failed to ensure PREROUTING chain exists", slog.String("error", err.Error()))
	}

	m.logger.Info("nftables nat table and PREROUTING chain ensured")
	return nil
}
