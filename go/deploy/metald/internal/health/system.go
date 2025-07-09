package health

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

// SystemInfo contains system information for health checks
type SystemInfo struct {
	Hostname string `json:"hostname"`
	CPU      CPU    `json:"cpu"`
	Memory   Memory `json:"memory"`
	Uptime   string `json:"uptime"`
}

// CPU contains CPU information
type CPU struct {
	Architecture string `json:"architecture"`
	Cores        int    `json:"cores"`
	Model        string `json:"model,omitempty"`
}

// Memory contains memory information in bytes
type Memory struct {
	Total     uint64  `json:"total_bytes"`
	Used      uint64  `json:"used_bytes"`
	Available uint64  `json:"available_bytes"`
	UsedPct   float64 `json:"used_percent"`
}

// GetSystemInfo collects current system information
func GetSystemInfo(ctx context.Context, startTime time.Time) (*SystemInfo, error) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	// Get CPU information
	cpu := CPU{ //nolint:exhaustruct // Model field is populated conditionally below if available
		Architecture: runtime.GOARCH,
		Cores:        runtime.NumCPU(),
	}

	// Try to get CPU model from /proc/cpuinfo on Linux
	if model := getCPUModel(); model != "" {
		cpu.Model = model
	}

	// Get memory information
	memory := getMemoryInfo()

	// Calculate uptime
	uptime := time.Since(startTime).String()

	return &SystemInfo{
		Hostname: hostname,
		CPU:      cpu,
		Memory:   memory,
		Uptime:   uptime,
	}, nil
}

// getCPUModel attempts to read CPU model from /proc/cpuinfo
func getCPUModel() string {
	// AIDEV-NOTE: This is Linux-specific, could be extended for other OSes
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return ""
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

// getMemoryInfo gets memory information using Go runtime stats
func getMemoryInfo() Memory {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// AIDEV-NOTE: This provides Go runtime memory stats, not system memory
	// For system memory, we'd need to read /proc/meminfo on Linux
	systemMem := getSystemMemory()
	if systemMem.Total > 0 {
		return systemMem
	}

	// Fallback to runtime memory stats
	return Memory{
		Total:     m.Sys,
		Used:      m.Alloc,
		Available: m.Sys - m.Alloc,
		UsedPct:   float64(m.Alloc) / float64(m.Sys) * 100,
	}
}

// getSystemMemory attempts to read system memory from /proc/meminfo
func getSystemMemory() Memory {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return Memory{} //exhaustruct:ignore
	}

	var total, available uint64
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "MemTotal:") {
			if _, err := fmt.Sscanf(line, "MemTotal: %d kB", &total); err != nil {
				continue
			}
			total *= 1024 // Convert to bytes
		} else if strings.HasPrefix(line, "MemAvailable:") {
			if _, err := fmt.Sscanf(line, "MemAvailable: %d kB", &available); err != nil {
				continue
			}
			available *= 1024 // Convert to bytes
		}
	}

	if total == 0 {
		return Memory{} //exhaustruct:ignore
	}

	used := total - available
	usedPct := float64(used) / float64(total) * 100

	return Memory{
		Total:     total,
		Used:      used,
		Available: available,
		UsedPct:   usedPct,
	}
}
