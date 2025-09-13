package jailer

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/metald/internal/config"
)

func TestNewJailer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg := &config.JailerConfig{
		ChrootBaseDir: "/tmp/test-jailer",
		UID:           1000,
		GID:           1000,
	}

	jailer := NewJailer(logger, cfg)
	assert.NotNil(t, jailer)
	assert.Equal(t, cfg, jailer.config)
}

func TestSetupChroot(t *testing.T) {
	// This test requires root privileges to create device nodes
	if os.Getuid() != 0 {
		t.Skip("Test requires root privileges")
	}

	tmpDir, err := os.MkdirTemp("", "jailer-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg := &config.JailerConfig{
		ChrootBaseDir: tmpDir,
		UID:           1000,
		GID:           1000,
	}

	jailer := NewJailer(logger, cfg)
	chrootPath := filepath.Join(tmpDir, "test-vm", "root")

	err = jailer.setupChroot(context.Background(), chrootPath)
	assert.NoError(t, err)

	// Verify directories exist
	assert.DirExists(t, chrootPath)
	assert.DirExists(t, filepath.Join(chrootPath, "dev"))
	assert.DirExists(t, filepath.Join(chrootPath, "dev/net"))
	assert.DirExists(t, filepath.Join(chrootPath, "run"))

	// Verify device nodes exist
	tunPath := filepath.Join(chrootPath, "dev/net/tun")
	kvmPath := filepath.Join(chrootPath, "dev/kvm")

	tunInfo, err := os.Stat(tunPath)
	assert.NoError(t, err)
	assert.True(t, tunInfo.Mode()&os.ModeDevice != 0, "tun should be a device")

	kvmInfo, err := os.Stat(kvmPath)
	assert.NoError(t, err)
	assert.True(t, kvmInfo.Mode()&os.ModeDevice != 0, "kvm should be a device")
}

func TestExecOptions(t *testing.T) {
	opts := &ExecOptions{ //nolint:exhaustruct // Test only sets required fields for validation
		VMId:             "test-vm",
		NetworkNamespace: "/run/netns/test-vm",
		SocketPath:       "/firecracker.sock",
		FirecrackerArgs:  []string{"--config-file", "config.json"},
	}

	assert.Equal(t, "test-vm", opts.VMId)
	assert.Equal(t, "/run/netns/test-vm", opts.NetworkNamespace)
	assert.Equal(t, "/firecracker.sock", opts.SocketPath)
	assert.Len(t, opts.FirecrackerArgs, 2)
}

// TestJoinNetworkNamespace tests network namespace joining
// This test requires root privileges to create network namespaces
func TestJoinNetworkNamespace(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("Test requires root privileges")
	}

	// Create a test network namespace
	tmpDir, err := os.MkdirTemp("", "netns-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// This would require actual network namespace creation
	// which is complex to test without full network setup
	t.Skip("Network namespace testing requires complex setup")
}

// TestDropPrivileges tests privilege dropping
// This test is dangerous to run as it actually drops privileges
func TestDropPrivileges(t *testing.T) {
	t.Skip("Privilege dropping test would affect the test process")
}

// Integration test placeholder
func TestIntegratedJailerWorkflow(t *testing.T) {
	t.Skip("Integration test requires full environment setup")

	// This would test:
	// 1. Setting up chroot
	// 2. Joining network namespace
	// 3. Dropping privileges
	// 4. Executing a test binary instead of firecracker
}

// AIDEV-NOTE: These tests cover the basic functionality of the integrated jailer
// More comprehensive tests would require:
// 1. Root privileges or specific capabilities
// 2. Network namespace creation utilities
// 3. A test binary to execute instead of firecracker
// 4. Integration with the actual VM creation workflow
