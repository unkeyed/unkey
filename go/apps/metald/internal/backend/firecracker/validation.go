//go:build linux
// +build linux

package firecracker

import (
	"fmt"
	"net"
	"regexp"
)

// validatePath validates a file path to ensure it's safe to use
func validatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Check for path traversal attempts
	if containsPathTraversal(path) {
		return fmt.Errorf("path contains directory traversal: %s", path)
	}

	return nil
}

// containsPathTraversal checks if a path contains directory traversal patterns
func containsPathTraversal(path string) bool {
	patterns := []string{
		"..",
		"..\\",
		"../",
		"..\\",
	}

	for _, pattern := range patterns {
		if regexp.MustCompile(pattern).MatchString(path) {
			return true
		}
	}
	return false
}

// validateMAC validates a MAC address format
func validateMAC(mac string) error {
	if mac == "" {
		return fmt.Errorf("MAC address cannot be empty")
	}

	// Standard MAC address format: XX:XX:XX:XX:XX:XX
	macRegex := regexp.MustCompile(`^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$`)
	if !macRegex.MatchString(mac) {
		return fmt.Errorf("invalid MAC address format: %s", mac)
	}

	return nil
}

// validateCIDR validates a CIDR notation
func validateCIDR(cidr string) error {
	_, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("invalid CIDR notation: %s", cidr)
	}
	return nil
}

// validateMemorySize validates memory size in bytes
func validateMemorySize(sizeBytes int64) error {
	const (
		minMemory = 128 * 1024 * 1024        // 128 MB minimum
		maxMemory = 512 * 1024 * 1024 * 1024 // 512 GB maximum
	)

	if sizeBytes < minMemory {
		return fmt.Errorf("memory size %d bytes is below minimum of %d bytes (128MB)", sizeBytes, minMemory)
	}

	if sizeBytes > maxMemory {
		return fmt.Errorf("memory size %d bytes exceeds maximum of %d bytes (512GB)", sizeBytes, maxMemory)
	}

	// Check if memory is a multiple of 1MB (Firecracker requirement)
	if sizeBytes%(1024*1024) != 0 {
		return fmt.Errorf("memory size must be a multiple of 1MB")
	}

	return nil
}

// validateVCPUCount validates the number of vCPUs
func validateVCPUCount(count int32) error {
	if count < 1 {
		return fmt.Errorf("vCPU count must be at least 1")
	}

	if count > 32 {
		return fmt.Errorf("vCPU count %d exceeds maximum of 32", count)
	}

	// Firecracker works best with power-of-2 vCPU counts
	if !isPowerOfTwo(int(count)) && count != 1 {
		// This is a warning, not an error
		// Log it but don't fail
	}

	return nil
}

// isPowerOfTwo checks if a number is a power of two
func isPowerOfTwo(n int) bool {
	return n > 0 && (n&(n-1)) == 0
}

// validateDiskPath validates a disk image path
func validateDiskPath(path string) error {
	if err := validatePath(path); err != nil {
		return err
	}

	// Check for supported disk image extensions
	validExtensions := []string{".ext4", ".img", ".raw", ".qcow2"}
	hasValidExt := false
	for _, ext := range validExtensions {
		if regexp.MustCompile(ext + "$").MatchString(path) {
			hasValidExt = true
			break
		}
	}

	if !hasValidExt {
		return fmt.Errorf("disk path %s does not have a supported extension (.ext4, .img, .raw, .qcow2)", path)
	}

	return nil
}
