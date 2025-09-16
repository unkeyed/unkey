package network

// setupVMNetworking creates and configures TAP and veth devices for a VM
func (m *Manager) setupTunTap(ip string) error {
	// // tap := &netlink.Tuntap{
	// // 	LinkAttrs: netlink.LinkAttrs{
	// // 		Name: deviceNames.TAP,
	// // 	},
	// // 	Mode: netlink.TUNTAP_MODE_TAP,
	// // }

	// // if err := netlink.LinkAdd(tap); err != nil {
	// // 	return fmt.Errorf("failed to create TAP device %s: %w", deviceNames.TAP, err)
	// // }

	// // tapLink, err := netlink.LinkByName(deviceNames.TAP)
	// // if err != nil {
	// // 	return fmt.Errorf("failed to get TAP device: %w", err)
	// // }

	// // Set TAP device up
	// if err := netlink.LinkSetUp(tap); err != nil {
	// 	return fmt.Errorf("failed to bring TAP device up: %w", err)
	// }

	// m.logger.Info("TAP device created and up", slog.String("tap", tapLink.Attrs().Name))

	return nil
}
