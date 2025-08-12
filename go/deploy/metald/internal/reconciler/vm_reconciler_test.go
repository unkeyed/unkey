package reconciler

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/unkeyed/unkey/go/deploy/metald/internal/database"
	metaldv1 "github.com/unkeyed/unkey/go/gen/proto/metal/vmprovisioner/v1"
)

// AIDEV-NOTE: Test suite to prevent config serialization format regressions
// These tests ensure VM configs are consistently handled as JSON for database storage

func TestHasCorruptedConfig(t *testing.T) {
	logger := slog.Default()
	reconciler := &VMReconciler{
		logger: logger,
	}
	ctx := context.Background()

	t.Run("valid JSON config should not be corrupted", func(t *testing.T) {
		// Create a valid VM config
		config := &metaldv1.VmConfig{
			Cpu: &metaldv1.CpuConfig{
				VcpuCount: 2,
			},
			Memory: &metaldv1.MemoryConfig{
				SizeBytes: 1024 * 1024 * 1024, // 1 GiB
			},
			Boot: &metaldv1.BootConfig{
				KernelPath: "/path/to/kernel",
			},
			Storage: []*metaldv1.StorageDevice{
				{
					Id:           "root",
					Path:         "/path/to/rootfs",
					IsRootDevice: true,
				},
			},
		}

		// Marshal to JSON (how it's stored in database)
		configBytes, err := json.Marshal(config)
		if err != nil {
			t.Fatalf("failed to marshal config: %v", err)
		}

		vm := &database.VM{
			ID:     "test-vm-1",
			Config: configBytes,
		}

		// Should not be detected as corrupted
		if reconciler.hasCorruptedConfig(ctx, vm) {
			t.Error("valid JSON config was incorrectly detected as corrupted")
		}
	})

	t.Run("actually corrupted data should be detected", func(t *testing.T) {
		vm := &database.VM{
			ID:     "test-vm-2",
			Config: []byte("this is not valid JSON data"),
		}

		// Should be detected as corrupted
		if !reconciler.hasCorruptedConfig(ctx, vm) {
			t.Error("corrupted config was not detected")
		}
	})

	t.Run("empty config should be detected as corrupted", func(t *testing.T) {
		vm := &database.VM{
			ID:     "test-vm-3",
			Config: []byte{},
		}

		// Empty config unmarshals to nil/zero values, which should be considered invalid
		if !reconciler.hasCorruptedConfig(ctx, vm) {
			t.Error("empty config was not detected as corrupted")
		}
	})

	t.Run("regression test: protobuf config should be detected as corrupted", func(t *testing.T) {
		// AIDEV-BUSINESS_RULE: This test prevents regression to protobuf format
		// VM configs must be JSON, not protobuf, for consistency with port_mappings

		// Use protobuf-like bytes that would fail JSON parsing (simulates old format)
		protobufLikeBytes := []byte{0x08, 0x02, 0x12, 0x08, 0x08, 0x80, 0x80, 0x80, 0x40}

		vm := &database.VM{
			ID:     "test-vm-4",
			Config: protobufLikeBytes,
		}

		// Protobuf format should be detected as corrupted (prevents regression)
		if !reconciler.hasCorruptedConfig(ctx, vm) {
			t.Error("protobuf-like config was not detected as corrupted - this indicates a regression to the old format")
		}
	})

	t.Run("malformed JSON should be detected as corrupted", func(t *testing.T) {
		// Invalid JSON that starts correctly but is truncated
		malformedJSON := []byte(`{"cpu": {"vcpu_count": 2}, "memory": {"size_bytes":`)

		vm := &database.VM{
			ID:     "test-vm-5",
			Config: malformedJSON,
		}

		// Should be detected as corrupted
		if !reconciler.hasCorruptedConfig(ctx, vm) {
			t.Error("malformed JSON was not detected as corrupted")
		}
	})
}

func TestConfigSerializationConsistency(t *testing.T) {
	// AIDEV-NOTE: This test ensures database storage and reconciler validation use same format

	config := &metaldv1.VmConfig{
		Cpu: &metaldv1.CpuConfig{
			VcpuCount: 4,
		},
		Memory: &metaldv1.MemoryConfig{
			SizeBytes: 2048 * 1024 * 1024, // 2 GiB
		},
		Boot: &metaldv1.BootConfig{
			KernelPath: "/boot/vmlinux",
			KernelArgs: "console=ttyS0 reboot=k",
		},
		Storage: []*metaldv1.StorageDevice{
			{
				Id:           "root",
				Path:         "/data/rootfs.ext4",
				IsRootDevice: true,
			},
		},
	}

	// Serialize the same way as database does (JSON)
	configBytes, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}

	// Deserialize the same way as reconciler validation does (JSON)
	var unmarshaled metaldv1.VmConfig
	if err := json.Unmarshal(configBytes, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal config: %v", err)
	}

	// Verify round-trip consistency
	if unmarshaled.Cpu.VcpuCount != config.Cpu.VcpuCount {
		t.Errorf("VcpuCount mismatch: got %d, want %d", unmarshaled.Cpu.VcpuCount, config.Cpu.VcpuCount)
	}
	if unmarshaled.Memory.SizeBytes != config.Memory.SizeBytes {
		t.Errorf("Memory SizeBytes mismatch: got %d, want %d", unmarshaled.Memory.SizeBytes, config.Memory.SizeBytes)
	}
	if unmarshaled.Boot.KernelPath != config.Boot.KernelPath {
		t.Errorf("KernelPath mismatch: got %s, want %s", unmarshaled.Boot.KernelPath, config.Boot.KernelPath)
	}
	if len(unmarshaled.Storage) != len(config.Storage) {
		t.Errorf("Storage length mismatch: got %d, want %d", len(unmarshaled.Storage), len(config.Storage))
	} else if len(unmarshaled.Storage) > 0 {
		if unmarshaled.Storage[0].Path != config.Storage[0].Path {
			t.Errorf("Storage Path mismatch: got %s, want %s", unmarshaled.Storage[0].Path, config.Storage[0].Path)
		}
	}
	if unmarshaled.Boot.KernelArgs != config.Boot.KernelArgs {
		t.Errorf("KernelArgs mismatch: got %s, want %s", unmarshaled.Boot.KernelArgs, config.Boot.KernelArgs)
	}
}

func TestJSONConsistencyWithPortMappings(t *testing.T) {
	// AIDEV-NOTE: This test ensures VM configs use same JSON format as port_mappings in database

	// Test that a config can be stored alongside port mappings (both as JSON)
	config := &metaldv1.VmConfig{
		Cpu: &metaldv1.CpuConfig{
			VcpuCount: 1,
		},
		Memory: &metaldv1.MemoryConfig{
			SizeBytes: 512 * 1024 * 1024, // 512 MiB
		},
	}

	// Serialize config as JSON
	configJSON, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal config as JSON: %v", err)
	}

	// Port mappings are also JSON in the database
	portMappingsJSON := `[{"container_port": 80, "host_port": 8080, "protocol": "tcp"}]`

	// Verify both can be parsed as valid JSON
	var configTest metaldv1.VmConfig
	if err := json.Unmarshal(configJSON, &configTest); err != nil {
		t.Errorf("config JSON is invalid: %v", err)
	}

	var portMappingsTest []map[string]interface{}
	if err := json.Unmarshal([]byte(portMappingsJSON), &portMappingsTest); err != nil {
		t.Errorf("port mappings JSON is invalid: %v", err)
	}

	// Both should be valid JSON and consistent in format
	if configTest.Cpu.VcpuCount != 1 {
		t.Errorf("config deserialization failed")
	}
	if len(portMappingsTest) != 1 {
		t.Errorf("port mappings deserialization failed")
	}
}
