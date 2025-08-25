package network

import (
	"fmt"
	"time"
)

// BridgeCapacityAlert represents a bridge capacity alert
type BridgeCapacityAlert struct {
	BridgeName       string        `json:"bridge_name"`
	CurrentVMs       int64         `json:"current_vms"`
	MaxVMs           int64         `json:"max_vms"`
	UtilizationRatio float64       `json:"utilization_ratio"`
	Severity         AlertSeverity `json:"severity"`
	Threshold        float64       `json:"threshold"`
	Message          string        `json:"message"`
}

// AlertSeverity represents the severity level of an alert
type AlertSeverity string

const (
	AlertInfo     AlertSeverity = "info"
	AlertWarning  AlertSeverity = "warning"
	AlertCritical AlertSeverity = "critical"
)

// updateBridgeStats updates VM count and activity time for a bridge
func (m *NetworkMetrics) updateBridgeStats(bridgeName string, vmCountDelta int64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	stats, exists := m.bridgeStats[bridgeName]
	if !exists {
		stats = &BridgeStats{
			BridgeName:   bridgeName,
			VMCount:      0,
			MaxVMs:       100, // Default capacity
			CreatedAt:    time.Now(),
			LastActivity: time.Now(),
			IsHealthy:    true,
		}
		m.bridgeStats[bridgeName] = stats
	}

	// Update VM count and activity time
	newVMCount := stats.VMCount + vmCountDelta
	if newVMCount < 0 {
		newVMCount = 0 // VM count can't go negative
	}

	stats.VMCount = newVMCount
	stats.LastActivity = time.Now()

	// Update health status based on capacity
	utilizationRatio := float64(stats.VMCount) / float64(stats.MaxVMs)
	stats.IsHealthy = utilizationRatio < 0.95 // Consider unhealthy at 95% capacity
}

// SetBridgeMaxVMs sets the maximum VMs for a bridge
func (m *NetworkMetrics) SetBridgeMaxVMs(bridgeName string, maxVMs int64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	stats, exists := m.bridgeStats[bridgeName]
	if !exists {
		stats = &BridgeStats{
			BridgeName:   bridgeName,
			VMCount:      0,
			MaxVMs:       maxVMs,
			CreatedAt:    time.Now(),
			LastActivity: time.Now(),
			IsHealthy:    true,
		}
		m.bridgeStats[bridgeName] = stats
	} else {
		stats.MaxVMs = maxVMs
	}
}

// GetBridgeStats returns current bridge statistics
func (m *NetworkMetrics) GetBridgeStats() map[string]*BridgeStats {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Return a copy to avoid concurrent access issues
	statsCopy := make(map[string]*BridgeStats)
	for name, stats := range m.bridgeStats {
		statsCopy[name] = &BridgeStats{
			BridgeName:   stats.BridgeName,
			VMCount:      stats.VMCount,
			MaxVMs:       stats.MaxVMs,
			CreatedAt:    stats.CreatedAt,
			LastActivity: stats.LastActivity,
			IsHealthy:    stats.IsHealthy,
			ErrorCount:   stats.ErrorCount,
		}
	}

	return statsCopy
}

// GetBridgeCapacityAlerts returns bridges that are approaching capacity
func (m *NetworkMetrics) GetBridgeCapacityAlerts() []BridgeCapacityAlert {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var alerts []BridgeCapacityAlert

	for _, stats := range m.bridgeStats {
		ratio := float64(stats.VMCount) / float64(stats.MaxVMs)

		var severity AlertSeverity
		var threshold float64

		switch {
		case ratio >= 0.95:
			severity = AlertCritical
			threshold = 0.95
		case ratio >= 0.90:
			severity = AlertWarning
			threshold = 0.90
		case ratio >= 0.80:
			severity = AlertInfo
			threshold = 0.80
		default:
			continue // No alert needed
		}

		alerts = append(alerts, BridgeCapacityAlert{
			BridgeName:       stats.BridgeName,
			CurrentVMs:       stats.VMCount,
			MaxVMs:           stats.MaxVMs,
			UtilizationRatio: ratio,
			Severity:         severity,
			Threshold:        threshold,
			Message:          m.formatCapacityAlertMessage(stats, ratio, severity),
		})
	}

	return alerts
}

// formatCapacityAlertMessage creates a human-readable alert message
func (m *NetworkMetrics) formatCapacityAlertMessage(stats *BridgeStats, ratio float64, severity AlertSeverity) string {
	utilizationPercent := int(ratio * 100)

	switch severity {
	case AlertCritical:
		return fmt.Sprintf("CRITICAL: Bridge %s is at %d%% capacity (%d/%d VMs). Immediate action required!",
			stats.BridgeName, utilizationPercent, stats.VMCount, stats.MaxVMs)
	case AlertWarning:
		return fmt.Sprintf("WARNING: Bridge %s is at %d%% capacity (%d/%d VMs). Consider load balancing or scaling.",
			stats.BridgeName, utilizationPercent, stats.VMCount, stats.MaxVMs)
	case AlertInfo:
		return fmt.Sprintf("INFO: Bridge %s utilization is %d%% (%d/%d VMs). Monitor for continued growth.",
			stats.BridgeName, utilizationPercent, stats.VMCount, stats.MaxVMs)
	default:
		return fmt.Sprintf("Bridge %s utilization: %d%% (%d/%d VMs)",
			stats.BridgeName, utilizationPercent, stats.VMCount, stats.MaxVMs)
	}
}
