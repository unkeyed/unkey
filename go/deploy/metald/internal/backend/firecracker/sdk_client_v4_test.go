package firecracker

import (
	"testing"

	"github.com/unkeyed/unkey/go/deploy/metald/internal/config"
)

func TestNewSDKClientV4(t *testing.T) {
	// Test that we can create a new SDK client
	jailerConfig := &config.JailerConfig{
			ChrootBaseDir:         "/tmp/test-jailer",
		UID:                   1000,
		GID:                   1000,
	}

	// This is a basic smoke test to ensure the constructor works
	// More comprehensive testing is done through integration tests
	client, err := NewSDKClientV4(nil, nil, nil, jailerConfig, "/tmp")
	if err != nil {
		t.Fatalf("Expected to create SDK client, got error: %v", err)
	}

	if client == nil {
		t.Fatal("Expected non-nil client")
	}

	// Verify the client has the expected configuration
	if client.jailerConfig != jailerConfig {
		t.Error("Expected jailer config to match")
	}

	if client.baseDir != "/tmp" {
		t.Error("Expected base directory to match")
	}
}

func TestCreateTestConfig(t *testing.T) {
	config := &config.JailerConfig{
			ChrootBaseDir:         "/tmp/test-jailer",
		UID:                   1000,
		GID:                   1000,
	}


	if config.ChrootBaseDir != "/tmp/test-jailer" {
		t.Error("Expected chroot base dir to match")
	}

	if config.UID != 1000 {
		t.Error("Expected UID to be 1000")
	}

	if config.GID != 1000 {
		t.Error("Expected GID to be 1000")
	}
}