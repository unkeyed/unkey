package network

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// MultiBridgeState represents the serializable state for persistence
type MultiBridgeState struct {
	Workspaces  map[string]*WorkspaceAllocation `json:"workspaces"`
	BridgeUsage map[int]map[string]bool         `json:"bridge_usage"`
	LastSaved   time.Time                       `json:"last_saved"`
	Checksum    string                          `json:"checksum"` // SHA256 checksum for integrity validation
}

// saveState persists the current state to disk
func (mbm *MultiBridgeManager) saveState() error {
	start := time.Now()

	mbm.logger.Debug("saving state to disk",
		slog.String("state_path", mbm.statePath),
		slog.Int("workspace_count", len(mbm.workspaces)),
	)

	// Create state directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(mbm.statePath), 0755); err != nil {
		mbm.logger.Error("failed to create state directory",
			slog.String("error", err.Error()),
			slog.String("directory", filepath.Dir(mbm.statePath)),
		)
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	state := &MultiBridgeState{
		Workspaces:  mbm.workspaces,
		BridgeUsage: mbm.bridgeUsage,
		LastSaved:   time.Now(),
	}

	// Calculate checksum of state content (excluding checksum field)
	checksum, err := mbm.calculateStateChecksum(state)
	if err != nil {
		mbm.logger.Error("failed to calculate state checksum",
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to calculate state checksum: %w", err)
	}
	state.Checksum = checksum

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		mbm.logger.Error("failed to marshal state",
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write to temporary file first, then atomic rename
	tmpPath := mbm.statePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		mbm.logger.Error("failed to write state file",
			slog.String("error", err.Error()),
			slog.String("temp_path", tmpPath),
		)
		return fmt.Errorf("failed to write state file: %w", err)
	}

	if err := os.Rename(tmpPath, mbm.statePath); err != nil {
		mbm.logger.Error("failed to rename state file",
			slog.String("error", err.Error()),
			slog.String("temp_path", tmpPath),
			slog.String("final_path", mbm.statePath),
		)
		return fmt.Errorf("failed to rename state file: %w", err)
	}

	mbm.logger.Debug("state saved successfully",
		slog.String("state_path", mbm.statePath),
		slog.Duration("duration", time.Since(start)),
		slog.Int("data_size_bytes", len(data)),
	)

	return nil
}

// loadState restores state from disk
func (mbm *MultiBridgeManager) loadState() error {
	if _, err := os.Stat(mbm.statePath); os.IsNotExist(err) {
		// No state file exists yet - this is fine for first run
		return nil
	}

	data, err := os.ReadFile(mbm.statePath)
	if err != nil {
		return fmt.Errorf("failed to read state file: %w", err)
	}

	var state MultiBridgeState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("failed to unmarshal state: %w", err)
	}

	// Verify checksum first to detect corruption
	if err := mbm.verifyStateChecksum(&state); err != nil {
		return fmt.Errorf("state file checksum verification failed: %w", err)
	}

	// Validate state integrity before applying
	if err := mbm.validateState(&state); err != nil {
		return fmt.Errorf("corrupted state file: %w", err)
	}

	// Restore state
	mbm.workspaces = state.Workspaces
	mbm.bridgeUsage = state.BridgeUsage

	// Initialize maps if they're nil
	if mbm.workspaces == nil {
		mbm.workspaces = make(map[string]*WorkspaceAllocation)
	}
	if mbm.bridgeUsage == nil {
		mbm.bridgeUsage = make(map[int]map[string]bool)
	}

	// Initialize nested maps in bridgeUsage
	for bridgeNum := range mbm.bridgeUsage {
		if mbm.bridgeUsage[bridgeNum] == nil {
			mbm.bridgeUsage[bridgeNum] = make(map[string]bool)
		}
	}

	return nil
}

// calculateStateChecksum computes SHA256 checksum of state content (excluding checksum field)
func (mbm *MultiBridgeManager) calculateStateChecksum(state *MultiBridgeState) (string, error) {
	// Create a copy of state without the checksum field for consistent hashing
	stateForHashing := &MultiBridgeState{
		Workspaces:  state.Workspaces,
		BridgeUsage: state.BridgeUsage,
		LastSaved:   state.LastSaved,
		// Checksum field is intentionally excluded
	}

	// Marshal to JSON for consistent byte representation
	data, err := json.Marshal(stateForHashing)
	if err != nil {
		return "", fmt.Errorf("failed to marshal state for checksum: %w", err)
	}

	// Calculate SHA256 hash
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// verifyStateChecksum verifies the integrity of loaded state using stored checksum
func (mbm *MultiBridgeManager) verifyStateChecksum(state *MultiBridgeState) error {
	// Skip verification if no checksum is present (legacy state files)
	if state.Checksum == "" {
		mbm.logger.Warn("state file has no checksum, skipping integrity verification",
			slog.String("state_path", mbm.statePath),
		)
		return nil
	}

	// Calculate expected checksum
	expectedChecksum, err := mbm.calculateStateChecksum(state)
	if err != nil {
		return fmt.Errorf("failed to calculate expected checksum: %w", err)
	}

	// Compare checksums
	if state.Checksum != expectedChecksum {
		mbm.logger.Error("state file integrity check failed",
			slog.String("state_path", mbm.statePath),
			slog.String("stored_checksum", state.Checksum),
			slog.String("calculated_checksum", expectedChecksum),
		)
		return fmt.Errorf("checksum mismatch: stored=%s, calculated=%s", state.Checksum, expectedChecksum)
	}

	mbm.logger.Debug("state file integrity verified",
		slog.String("state_path", mbm.statePath),
		slog.String("checksum", state.Checksum),
	)

	return nil
}
