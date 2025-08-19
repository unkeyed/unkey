package health

import (
	"context"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSystemInfo(t *testing.T) {
	startTime := time.Now().Add(-1 * time.Hour)
	ctx := context.Background()

	systemInfo, err := GetSystemInfo(ctx, startTime)
	require.NoError(t, err)
	require.NotNil(t, systemInfo)

	// Test hostname (should not be empty unless there's an error)
	assert.NotEmpty(t, systemInfo.Hostname)

	// Test CPU info
	assert.Equal(t, runtime.GOARCH, systemInfo.CPU.Architecture)
	assert.Equal(t, runtime.NumCPU(), systemInfo.CPU.Cores)
	assert.GreaterOrEqual(t, systemInfo.CPU.Cores, 1)

	// Test memory info
	assert.Greater(t, systemInfo.Memory.Total, uint64(0))
	assert.GreaterOrEqual(t, systemInfo.Memory.Used, uint64(0))
	assert.GreaterOrEqual(t, systemInfo.Memory.Available, uint64(0))
	assert.GreaterOrEqual(t, systemInfo.Memory.UsedPct, 0.0)
	assert.LessOrEqual(t, systemInfo.Memory.UsedPct, 100.0)

	// Test uptime (should be positive duration string)
	assert.NotEmpty(t, systemInfo.Uptime)
	assert.True(t, strings.Contains(systemInfo.Uptime, "h") || strings.Contains(systemInfo.Uptime, "m") || strings.Contains(systemInfo.Uptime, "s"))
}

func TestGetSystemInfo_Uptime(t *testing.T) {
	// Test with start time 2 hours ago
	startTime := time.Now().Add(-2 * time.Hour)
	ctx := context.Background()

	systemInfo, err := GetSystemInfo(ctx, startTime)
	require.NoError(t, err)

	// Uptime should reflect approximately 2 hours
	assert.Contains(t, systemInfo.Uptime, "h")
}

func TestCPU_Structure(t *testing.T) {
	cpu := CPU{
		Architecture: "amd64",
		Cores:        8,
		Model:        "Intel Core i7",
	}

	assert.Equal(t, "amd64", cpu.Architecture)
	assert.Equal(t, 8, cpu.Cores)
	assert.Equal(t, "Intel Core i7", cpu.Model)
}

func TestMemory_Structure(t *testing.T) {
	memory := Memory{
		Total:     8589934592, // 8GB
		Used:      2147483648, // 2GB
		Available: 6442450944, // 6GB
		UsedPct:   25.0,
	}

	assert.Equal(t, uint64(8589934592), memory.Total)
	assert.Equal(t, uint64(2147483648), memory.Used)
	assert.Equal(t, uint64(6442450944), memory.Available)
	assert.Equal(t, 25.0, memory.UsedPct)
}

func TestSystemInfo_Structure(t *testing.T) {
	systemInfo := SystemInfo{
		Hostname: "test-host",
		CPU: CPU{
			Architecture: "amd64",
			Cores:        4,
			Model:        "Test CPU",
		},
		Memory: Memory{
			Total:     4294967296,
			Used:      1073741824,
			Available: 3221225472,
			UsedPct:   25.0,
		},
		Uptime: "2h30m15s",
	}

	assert.Equal(t, "test-host", systemInfo.Hostname)
	assert.Equal(t, "amd64", systemInfo.CPU.Architecture)
	assert.Equal(t, 4, systemInfo.CPU.Cores)
	assert.Equal(t, "Test CPU", systemInfo.CPU.Model)
	assert.Equal(t, uint64(4294967296), systemInfo.Memory.Total)
	assert.Equal(t, "2h30m15s", systemInfo.Uptime)
}

func TestGetCPUModel(t *testing.T) {
	// This test is environment-dependent and might not work on all systems
	// We'll test the function but accept empty results on non-Linux systems
	model := getCPUModel()

	// On Linux systems with /proc/cpuinfo, we might get a model
	// On other systems or containers, this might be empty
	// Both are valid outcomes
	if runtime.GOOS == "linux" {
		// Model might be available on Linux, but not guaranteed in containers
		t.Logf("CPU Model detected: %q", model)
	} else {
		// On non-Linux systems, this will likely be empty
		assert.Equal(t, "", model)
	}
}

func TestGetMemoryInfo(t *testing.T) {
	memory := getMemoryInfo()

	// Basic validation - memory values should be reasonable
	assert.Greater(t, memory.Total, uint64(0))
	assert.GreaterOrEqual(t, memory.Used, uint64(0))
	assert.GreaterOrEqual(t, memory.Available, uint64(0))
	assert.GreaterOrEqual(t, memory.UsedPct, 0.0)
	assert.LessOrEqual(t, memory.UsedPct, 100.0)

	// Total should be greater than or equal to used
	assert.GreaterOrEqual(t, memory.Total, memory.Used)
}

func TestGetSystemMemory(t *testing.T) {
	memory := getSystemMemory()

	// This test is Linux-specific since it reads /proc/meminfo
	if runtime.GOOS == "linux" {
		if _, err := os.Stat("/proc/meminfo"); err == nil {
			// /proc/meminfo exists, we should get some data
			assert.Greater(t, memory.Total, uint64(0))
			assert.GreaterOrEqual(t, memory.Used, uint64(0))
			assert.GreaterOrEqual(t, memory.Available, uint64(0))
			assert.GreaterOrEqual(t, memory.UsedPct, 0.0)
			assert.LessOrEqual(t, memory.UsedPct, 100.0)
		}
	}
	// On non-Linux systems or when /proc/meminfo is not available,
	// the function returns zero values, which is expected behavior
}

func TestMemoryCalculations(t *testing.T) {
	// Test memory percentage calculation logic
	tests := []struct {
		name      string
		total     uint64
		available uint64
		wantUsed  uint64
		wantPct   float64
	}{
		{
			name:      "50% usage",
			total:     1000,
			available: 500,
			wantUsed:  500,
			wantPct:   50.0,
		},
		{
			name:      "25% usage",
			total:     2000,
			available: 1500,
			wantUsed:  500,
			wantPct:   25.0,
		},
		{
			name:      "0% usage",
			total:     1000,
			available: 1000,
			wantUsed:  0,
			wantPct:   0.0,
		},
		{
			name:      "100% usage",
			total:     1000,
			available: 0,
			wantUsed:  1000,
			wantPct:   100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			used := tt.total - tt.available
			usedPct := float64(used) / float64(tt.total) * 100

			assert.Equal(t, tt.wantUsed, used)
			assert.Equal(t, tt.wantPct, usedPct)
		})
	}
}

func TestGetSystemInfo_ErrorHandling(t *testing.T) {
	// Test with a very recent start time (basically now)
	startTime := time.Now()
	ctx := context.Background()

	systemInfo, err := GetSystemInfo(ctx, startTime)
	require.NoError(t, err)
	require.NotNil(t, systemInfo)

	// Even if hostname fails, the function should not return an error
	// It should use "unknown" as fallback
	assert.NotEmpty(t, systemInfo.Hostname)

	// CPU info should still be populated from runtime
	assert.Equal(t, runtime.GOARCH, systemInfo.CPU.Architecture)
	assert.Equal(t, runtime.NumCPU(), systemInfo.CPU.Cores)

	// Memory should be populated (either from system or runtime)
	assert.Greater(t, systemInfo.Memory.Total, uint64(0))

	// Uptime should be very small but formatted as string
	assert.NotEmpty(t, systemInfo.Uptime)
}
