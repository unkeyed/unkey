package database

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	metaldv1 "github.com/unkeyed/unkey/go/gen/proto/metal/vmprovisioner/v1"
)

func setupTestDB(t *testing.T) *Database {
	tempDir := t.TempDir()
	db, err := New(tempDir)
	require.NoError(t, err)
	require.NotNil(t, db)
	t.Cleanup(func() {
		db.Close()
	})
	return db
}

func TestNewVMRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewVMRepository(db)

	assert.NotNil(t, repo)
	assert.Equal(t, db, repo.db)
	assert.NotNil(t, repo.logger)
}

func TestVMRepository_CreateVM(t *testing.T) {
	db := setupTestDB(t)
	repo := NewVMRepository(db)

	vmID := "test-vm-1"
	customerID := "customer-1"
	config := &metaldv1.VmConfig{
		Cpu: &metaldv1.CpuConfig{
			VcpuCount: 2,
		},
		Memory: &metaldv1.MemoryConfig{
			SizeBytes: 1024 * 1024 * 1024, // 1GB
		},
		Boot: &metaldv1.BootConfig{
			KernelPath: "/path/to/kernel",
		},
		Metadata: map[string]string{
			"id": vmID,
		},
	}
	state := metaldv1.VmState_VM_STATE_CREATED

	err := repo.CreateVM(vmID, customerID, config, state)
	assert.NoError(t, err)

	// Verify VM was created
	vm, err := repo.GetVM(vmID)
	require.NoError(t, err)
	assert.Equal(t, vmID, vm.ID)
	assert.Equal(t, customerID, vm.CustomerID)
	assert.Equal(t, state, vm.State)
	assert.Equal(t, "[]", vm.PortMappings)
	assert.Nil(t, vm.ProcessID)
	assert.Nil(t, vm.DeletedAt)
}

func TestVMRepository_CreateVMWithContext(t *testing.T) {
	db := setupTestDB(t)
	repo := NewVMRepository(db)

	ctx := context.Background()
	vmID := "test-vm-context"
	customerID := "customer-context"
	config := &metaldv1.VmConfig{
		Cpu: &metaldv1.CpuConfig{
			VcpuCount: 4,
		},
		Memory: &metaldv1.MemoryConfig{
			SizeBytes: 2048 * 1024 * 1024, // 2GB
		},
		Metadata: map[string]string{
			"id": vmID,
		},
	}
	state := metaldv1.VmState_VM_STATE_RUNNING

	err := repo.CreateVMWithContext(ctx, vmID, customerID, config, state)
	assert.NoError(t, err)

	// Verify creation
	vm, err := repo.GetVMWithContext(ctx, vmID)
	require.NoError(t, err)
	assert.Equal(t, vmID, vm.ID)
	assert.Equal(t, customerID, vm.CustomerID)
	assert.Equal(t, state, vm.State)
}

func TestVMRepository_GetVM_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewVMRepository(db)

	vm, err := repo.GetVM("nonexistent-vm")
	assert.Error(t, err)
	assert.Nil(t, vm)
	assert.Contains(t, err.Error(), "VM not found")
}

func TestVMRepository_UpdateVMState(t *testing.T) {
	db := setupTestDB(t)
	repo := NewVMRepository(db)

	// Create initial VM
	vmID := "test-vm-update"
	customerID := "customer-update"
	config := &metaldv1.VmConfig{
		Cpu: &metaldv1.CpuConfig{
			VcpuCount: 1,
		},
		Memory: &metaldv1.MemoryConfig{
			SizeBytes: 512 * 1024 * 1024, // 512MB
		},
		Metadata: map[string]string{
			"id": vmID,
		},
	}
	initialState := metaldv1.VmState_VM_STATE_CREATED

	err := repo.CreateVM(vmID, customerID, config, initialState)
	require.NoError(t, err)

	// Update state
	newState := metaldv1.VmState_VM_STATE_RUNNING
	processID := "12345"

	err = repo.UpdateVMState(vmID, newState, &processID)
	assert.NoError(t, err)

	// Verify update
	vm, err := repo.GetVM(vmID)
	require.NoError(t, err)
	assert.Equal(t, newState, vm.State)
	assert.NotNil(t, vm.ProcessID)
	assert.Equal(t, processID, *vm.ProcessID)
}

func TestVMRepository_UpdateVMState_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewVMRepository(db)

	err := repo.UpdateVMState("nonexistent-vm", metaldv1.VmState_VM_STATE_RUNNING, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "VM not found or already deleted")
}

func TestVMRepository_DeleteVM(t *testing.T) {
	db := setupTestDB(t)
	repo := NewVMRepository(db)

	// Create VM
	vmID := "test-vm-delete"
	customerID := "customer-delete"
	config := &metaldv1.VmConfig{
		Metadata: map[string]string{
			"id": vmID,
		},
	}
	state := metaldv1.VmState_VM_STATE_CREATED

	err := repo.CreateVM(vmID, customerID, config, state)
	require.NoError(t, err)

	// Delete VM
	err = repo.DeleteVM(vmID)
	assert.NoError(t, err)

	// Verify VM is not found after deletion
	vm, err := repo.GetVM(vmID)
	assert.Error(t, err)
	assert.Nil(t, vm)
	assert.Contains(t, err.Error(), "VM not found")
}

func TestVMRepository_DeleteVM_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewVMRepository(db)

	err := repo.DeleteVM("nonexistent-vm")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "VM not found or already deleted")
}

func TestVMRepository_ListVMs(t *testing.T) {
	db := setupTestDB(t)
	repo := NewVMRepository(db)

	// Create multiple VMs
	customerID1 := "customer-1"
	customerID2 := "customer-2"

	vms := []struct {
		id         string
		customerID string
		state      metaldv1.VmState
	}{
		{"vm-1", customerID1, metaldv1.VmState_VM_STATE_CREATED},
		{"vm-2", customerID1, metaldv1.VmState_VM_STATE_RUNNING},
		{"vm-3", customerID2, metaldv1.VmState_VM_STATE_CREATED},
		{"vm-4", customerID2, metaldv1.VmState_VM_STATE_SHUTDOWN},
	}

	for _, vm := range vms {
		config := &metaldv1.VmConfig{
			Metadata: map[string]string{
				"id": vm.id,
			},
		}
		err := repo.CreateVM(vm.id, vm.customerID, config, vm.state)
		require.NoError(t, err)
	}

	// List all VMs
	allVMs, err := repo.ListVMs(nil, nil, 0, 0)
	require.NoError(t, err)
	assert.Len(t, allVMs, 4)

	// List VMs by customer
	customer1VMs, err := repo.ListVMs(&customerID1, nil, 0, 0)
	require.NoError(t, err)
	assert.Len(t, customer1VMs, 2)

	// List VMs by state
	createdState := []metaldv1.VmState{metaldv1.VmState_VM_STATE_CREATED}
	createdVMs, err := repo.ListVMs(nil, createdState, 0, 0)
	require.NoError(t, err)
	assert.Len(t, createdVMs, 2)

	// List with limit
	limitedVMs, err := repo.ListVMs(nil, nil, 2, 0)
	require.NoError(t, err)
	assert.Len(t, limitedVMs, 2)

	// List with offset
	offsetVMs, err := repo.ListVMs(nil, nil, 2, 2)
	require.NoError(t, err)
	assert.Len(t, offsetVMs, 2)
}

func TestVMRepository_CountVMs(t *testing.T) {
	db := setupTestDB(t)
	repo := NewVMRepository(db)

	customerID := "customer-count"

	// Create test VMs
	for i := 0; i < 5; i++ {
		vmID := "vm-" + string(rune('1'+i))
		config := &metaldv1.VmConfig{
			Metadata: map[string]string{
				"id": vmID,
			},
		}
		state := metaldv1.VmState_VM_STATE_CREATED
		if i%2 == 0 {
			state = metaldv1.VmState_VM_STATE_RUNNING
		}
		err := repo.CreateVM(vmID, customerID, config, state)
		require.NoError(t, err)
	}

	// Count all VMs
	totalCount, err := repo.CountVMs(nil, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(5), totalCount)

	// Count by customer
	customerCount, err := repo.CountVMs(&customerID, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(5), customerCount)

	// Count by state
	runningState := []metaldv1.VmState{metaldv1.VmState_VM_STATE_RUNNING}
	runningCount, err := repo.CountVMs(nil, runningState)
	require.NoError(t, err)
	assert.Equal(t, int64(3), runningCount) // vm-1, vm-3, vm-5
}

func TestVMRepository_UpdateVMPortMappings(t *testing.T) {
	db := setupTestDB(t)
	repo := NewVMRepository(db)

	// Create VM
	vmID := "test-vm-ports"
	customerID := "customer-ports"
	config := &metaldv1.VmConfig{
		Metadata: map[string]string{
			"id": vmID,
		},
	}
	state := metaldv1.VmState_VM_STATE_CREATED

	err := repo.CreateVM(vmID, customerID, config, state)
	require.NoError(t, err)

	// Update port mappings
	portMappings := `[{"host_port": 8080, "guest_port": 80}, {"host_port": 8443, "guest_port": 443}]`

	err = repo.UpdateVMPortMappings(vmID, portMappings)
	assert.NoError(t, err)

	// Verify update
	vm, err := repo.GetVM(vmID)
	require.NoError(t, err)
	assert.Equal(t, portMappings, vm.PortMappings)
}

func TestVMRepository_ListVMsByCustomerWithContext(t *testing.T) {
	db := setupTestDB(t)
	repo := NewVMRepository(db)

	ctx := context.Background()
	customerID := "customer-list-context"

	// Create VMs with different configs
	configs := []*metaldv1.VmConfig{
		{
			Cpu:      &metaldv1.CpuConfig{VcpuCount: 2},
			Memory:   &metaldv1.MemoryConfig{SizeBytes: 1024 * 1024 * 1024},
			Metadata: map[string]string{"id": "vm-1"},
		},
		{
			Cpu:      &metaldv1.CpuConfig{VcpuCount: 4},
			Memory:   &metaldv1.MemoryConfig{SizeBytes: 2048 * 1024 * 1024},
			Metadata: map[string]string{"id": "vm-2"},
		},
	}

	for _, config := range configs {
		vmID := config.Metadata["id"]
		err := repo.CreateVMWithContext(ctx, vmID, customerID, config, metaldv1.VmState_VM_STATE_CREATED)
		require.NoError(t, err)
	}

	// List VMs with parsed configs
	vms, err := repo.ListVMsByCustomerWithContext(ctx, customerID)
	require.NoError(t, err)
	assert.Len(t, vms, 2)

	// Verify parsed configs are populated
	for _, vm := range vms {
		assert.NotNil(t, vm.ParsedConfig)
		assert.NotNil(t, vm.ParsedConfig.Cpu)
		assert.NotNil(t, vm.ParsedConfig.Memory)
		assert.Greater(t, vm.ParsedConfig.Cpu.VcpuCount, int32(0))
		assert.Greater(t, vm.ParsedConfig.Memory.SizeBytes, int64(0))
		assert.NotEmpty(t, vm.ParsedConfig.Metadata["id"])
	}
}

func TestVMRepository_ListAllVMsWithContext(t *testing.T) {
	db := setupTestDB(t)
	repo := NewVMRepository(db)

	ctx := context.Background()

	// Create test VMs
	for i := 1; i <= 3; i++ {
		vmID := "vm-all-" + string(rune('0'+i))
		customerID := "customer-" + string(rune('0'+i))
		config := &metaldv1.VmConfig{
			Metadata: map[string]string{
				"id": vmID,
			},
		}
		err := repo.CreateVMWithContext(ctx, vmID, customerID, config, metaldv1.VmState_VM_STATE_CREATED)
		require.NoError(t, err)
	}

	// List all VMs
	allVMs, err := repo.ListAllVMsWithContext(ctx)
	require.NoError(t, err)
	assert.Len(t, allVMs, 3)

	// Verify ordering (should be by created_at DESC)
	for i := 0; i < len(allVMs)-1; i++ {
		assert.True(t, allVMs[i].CreatedAt.After(allVMs[i+1].CreatedAt) ||
			allVMs[i].CreatedAt.Equal(allVMs[i+1].CreatedAt))
	}
}

func TestVMRepository_UpdateVMStateWithContextInt(t *testing.T) {
	db := setupTestDB(t)
	repo := NewVMRepository(db)

	ctx := context.Background()

	// Create VM
	vmID := "test-vm-int-state"
	customerID := "customer-int-state"
	config := &metaldv1.VmConfig{
		Metadata: map[string]string{
			"id": vmID,
		},
	}

	err := repo.CreateVMWithContext(ctx, vmID, customerID, config, metaldv1.VmState_VM_STATE_CREATED)
	require.NoError(t, err)

	// Update with int state
	newState := int(metaldv1.VmState_VM_STATE_RUNNING)
	err = repo.UpdateVMStateWithContextInt(ctx, vmID, newState)
	assert.NoError(t, err)

	// Verify update
	vm, err := repo.GetVMWithContext(ctx, vmID)
	require.NoError(t, err)
	assert.Equal(t, metaldv1.VmState_VM_STATE_RUNNING, vm.State)
}

func TestVMRepository_UpdateVMStateWithContextInt_Overflow(t *testing.T) {
	db := setupTestDB(t)
	repo := NewVMRepository(db)

	ctx := context.Background()

	// Create VM
	vmID := "test-vm-overflow"
	customerID := "customer-overflow"
	config := &metaldv1.VmConfig{
		Metadata: map[string]string{
			"id": vmID,
		},
	}

	err := repo.CreateVMWithContext(ctx, vmID, customerID, config, metaldv1.VmState_VM_STATE_CREATED)
	require.NoError(t, err)

	// Test overflow protection
	largeState := int(2147483648) // Larger than max int32
	err = repo.UpdateVMStateWithContextInt(ctx, vmID, largeState)
	assert.NoError(t, err) // Should not error, value should be clamped

	// Test underflow protection
	smallState := int(-2147483649) // Smaller than min int32
	err = repo.UpdateVMStateWithContextInt(ctx, vmID, smallState)
	assert.NoError(t, err) // Should not error, value should be clamped
}

func TestVM_GetVMConfig(t *testing.T) {
	config := &metaldv1.VmConfig{
		Cpu: &metaldv1.CpuConfig{
			VcpuCount: 4,
		},
		Memory: &metaldv1.MemoryConfig{
			SizeBytes: 2048 * 1024 * 1024, // 2GB
		},
		Boot: &metaldv1.BootConfig{
			KernelPath: "/path/to/kernel",
		},
		Metadata: map[string]string{
			"id": "test-vm",
		},
	}

	configBytes, err := json.Marshal(config)
	require.NoError(t, err)

	vm := &VM{
		ID:     "test-vm",
		Config: configBytes,
	}

	parsedConfig, err := vm.GetVMConfig()
	require.NoError(t, err)
	assert.Equal(t, config.Metadata["id"], parsedConfig.Metadata["id"])
	assert.Equal(t, config.Cpu.VcpuCount, parsedConfig.Cpu.VcpuCount)
	assert.Equal(t, config.Memory.SizeBytes, parsedConfig.Memory.SizeBytes)
	assert.Equal(t, config.Boot.KernelPath, parsedConfig.Boot.KernelPath)
}

func TestVM_GetVMConfig_InvalidJSON(t *testing.T) {
	vm := &VM{
		ID:     "test-vm",
		Config: []byte("invalid json"),
	}

	parsedConfig, err := vm.GetVMConfig()
	assert.Error(t, err)
	assert.Nil(t, parsedConfig)
	assert.Contains(t, err.Error(), "failed to unmarshal VM config")
}

func TestVMRepository_Integration(t *testing.T) {
	db := setupTestDB(t)
	repo := NewVMRepository(db)

	ctx := context.Background()

	// Full integration test: create, update, list, delete
	vmID := "integration-vm"
	customerID := "integration-customer"
	config := &metaldv1.VmConfig{
		Cpu: &metaldv1.CpuConfig{
			VcpuCount: 8,
		},
		Memory: &metaldv1.MemoryConfig{
			SizeBytes: 4096 * 1024 * 1024, // 4GB
		},
		Boot: &metaldv1.BootConfig{
			KernelPath: "/boot/vmlinux",
		},
		Storage: []*metaldv1.StorageDevice{
			{
				Id:           "storage-1",
				Path:         "/images/rootfs.ext4",
				IsRootDevice: true,
			},
		},
		Metadata: map[string]string{
			"id": vmID,
		},
	}

	// Create
	err := repo.CreateVMWithContext(ctx, vmID, customerID, config, metaldv1.VmState_VM_STATE_CREATED)
	require.NoError(t, err)

	// Update state
	processID := "integration-process-123"
	err = repo.UpdateVMStateWithContext(ctx, vmID, metaldv1.VmState_VM_STATE_RUNNING, &processID)
	require.NoError(t, err)

	// Update port mappings
	portMappings := `[{"host_port": 9090, "guest_port": 8080}]`
	err = repo.UpdateVMPortMappingsWithContext(ctx, vmID, portMappings)
	require.NoError(t, err)

	// Retrieve and verify
	vm, err := repo.GetVMWithContext(ctx, vmID)
	require.NoError(t, err)
	assert.Equal(t, vmID, vm.ID)
	assert.Equal(t, customerID, vm.CustomerID)
	assert.Equal(t, metaldv1.VmState_VM_STATE_RUNNING, vm.State)
	assert.NotNil(t, vm.ProcessID)
	assert.Equal(t, processID, *vm.ProcessID)
	assert.Equal(t, portMappings, vm.PortMappings)

	// List by customer
	customerVMs, err := repo.ListVMsByCustomerWithContext(ctx, customerID)
	require.NoError(t, err)
	assert.Len(t, customerVMs, 1)
	assert.Equal(t, vmID, customerVMs[0].ID)

	// Count
	count, err := repo.CountVMs(&customerID, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Delete
	err = repo.DeleteVMWithContext(ctx, vmID)
	require.NoError(t, err)

	// Verify deletion
	_, err = repo.GetVMWithContext(ctx, vmID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "VM not found")

	// Count after deletion
	count, err = repo.CountVMs(&customerID, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}
