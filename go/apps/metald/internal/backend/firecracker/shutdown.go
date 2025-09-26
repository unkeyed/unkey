//go:build linux
// +build linux

package firecracker

import "context"

// ShutdownVM gracefully shuts down a VM
func (c *Client) ShutdownVM(ctx context.Context, vmID string) error {
	return c.ShutdownVMWithOptions(ctx, vmID, false, 30)
}
