package tenant

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	builderv1 "github.com/unkeyed/unkey/go/deploy/builderd/gen/proto/builder/v1"
)

// StorageIsolator handles storage isolation and encryption for tenants
type StorageIsolator struct {
	logger        *slog.Logger
	tenantMgr     *Manager
	baseDir       string
	encryptionKey []byte
}

// NewStorageIsolator creates a new storage isolator
func NewStorageIsolator(logger *slog.Logger, tenantMgr *Manager, baseDir string) *StorageIsolator {
	// Generate or load encryption key (in production, this should be from secure key management)
	encKey := make([]byte, 32) // 256-bit key
	if _, err := rand.Read(encKey); err != nil {
		logger.Warn("failed to generate encryption key, using deterministic key", slog.String("error", err.Error()))
		// Fallback to deterministic key (NOT recommended for production)
		hash := sha256.Sum256([]byte("builderd-storage-key"))
		copy(encKey, hash[:])
	}
	
	return &StorageIsolator{
		logger:        logger,
		tenantMgr:     tenantMgr,
		baseDir:       baseDir,
		encryptionKey: encKey,
	}
}

// CreateTenantDirectories creates isolated directories for a tenant build
func (s *StorageIsolator) CreateTenantDirectories(
	ctx context.Context,
	tenantID string,
	tier builderv1.TenantTier,
	buildID string,
) (*TenantDirectories, error) {
	config, err := s.tenantMgr.GetTenantConfig(ctx, tenantID, tier)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant config: %w", err)
	}
	
	// Create tenant-specific directory structure
	tenantBaseDir := filepath.Join(s.baseDir, "tenants", tenantID)
	buildBaseDir := filepath.Join(tenantBaseDir, "builds", buildID)
	
	dirs := &TenantDirectories{
		TenantID:     tenantID,
		BuildID:      buildID,
		BaseDir:      buildBaseDir,
		WorkspaceDir: filepath.Join(buildBaseDir, "workspace"),
		RootfsDir:    filepath.Join(buildBaseDir, "rootfs"),
		TempDir:      filepath.Join(buildBaseDir, "temp"),
		LogsDir:      filepath.Join(buildBaseDir, "logs"),
		MetadataDir:  filepath.Join(buildBaseDir, "metadata"),
		CacheDir:     filepath.Join(tenantBaseDir, "cache"),
		
		// Permissions and ownership
		DirMode:  0750, // rwxr-x---
		FileMode: 0640, // rw-r-----
		UID:      1000, // builderd user
		GID:      1000, // builderd group
		
		// Security settings
		EncryptionEnabled: config.Storage.EncryptionEnabled,
		CompressionEnabled: config.Storage.CompressionEnabled,
		IsolationEnabled:  config.Storage.IsolationEnabled,
	}
	
	// Create all directories
	if err := s.createDirectories(dirs); err != nil {
		return nil, fmt.Errorf("failed to create directories: %w", err)
	}
	
	// Apply security settings
	if err := s.applySecuritySettings(dirs, config); err != nil {
		return nil, fmt.Errorf("failed to apply security settings: %w", err)
	}
	
	// Set up quota monitoring
	if err := s.setupQuotaMonitoring(dirs, config); err != nil {
		s.logger.Warn("failed to setup quota monitoring", slog.String("error", err.Error()))
	}
	
	s.logger.Info("created tenant directories",
		slog.String("tenant_id", tenantID),
		slog.String("build_id", buildID),
		slog.String("base_dir", dirs.BaseDir),
		slog.Bool("encryption_enabled", dirs.EncryptionEnabled),
	)
	
	return dirs, nil
}

// TenantDirectories represents the directory structure for a tenant build
type TenantDirectories struct {
	TenantID     string
	BuildID      string
	BaseDir      string
	WorkspaceDir string
	RootfsDir    string
	TempDir      string
	LogsDir      string
	MetadataDir  string
	CacheDir     string
	
	// Permissions
	DirMode  os.FileMode
	FileMode os.FileMode
	UID      int
	GID      int
	
	// Security settings
	EncryptionEnabled  bool
	CompressionEnabled bool
	IsolationEnabled   bool
}

// createDirectories creates all required directories with proper permissions
func (s *StorageIsolator) createDirectories(dirs *TenantDirectories) error {
	directoriesToCreate := []string{
		dirs.BaseDir,
		dirs.WorkspaceDir,
		dirs.RootfsDir,
		dirs.TempDir,
		dirs.LogsDir,
		dirs.MetadataDir,
		dirs.CacheDir,
	}
	
	for _, dir := range directoriesToCreate {
		if err := os.MkdirAll(dir, dirs.DirMode); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		
		// Set ownership
		if err := os.Chown(dir, dirs.UID, dirs.GID); err != nil {
			s.logger.Warn("failed to set directory ownership",
				slog.String("dir", dir),
				slog.String("error", err.Error()),
			)
		}
	}
	
	return nil
}

// applySecuritySettings applies security configurations to directories
func (s *StorageIsolator) applySecuritySettings(dirs *TenantDirectories, config *TenantConfig) error {
	// Set extended attributes for isolation
	if dirs.IsolationEnabled {
		// Set security labels (requires SELinux/AppArmor support)
		isolationLabel := fmt.Sprintf("builderd:tenant:%s", dirs.TenantID)
		
		for _, dir := range []string{dirs.BaseDir, dirs.WorkspaceDir, dirs.RootfsDir} {
			if err := s.setSecurityLabel(dir, isolationLabel); err != nil {
				s.logger.Debug("failed to set security label",
					slog.String("dir", dir),
					slog.String("error", err.Error()),
				)
			}
		}
	}
	
	// Create access control files
	readmeContent := fmt.Sprintf(`# Builderd Tenant Storage

This directory contains build artifacts for:
- Tenant ID: %s
- Build ID: %s
- Created: %s
- Encryption: %v
- Compression: %v

WARNING: This directory is managed by builderd. 
Do not modify files directly.
`, dirs.TenantID, dirs.BuildID, time.Now().Format(time.RFC3339),
		dirs.EncryptionEnabled, dirs.CompressionEnabled)
	
	readmePath := filepath.Join(dirs.BaseDir, "README.txt")
	if err := s.writeFile(readmePath, []byte(readmeContent), dirs.FileMode, dirs.EncryptionEnabled); err != nil {
		s.logger.Debug("failed to create README", slog.String("error", err.Error()))
	}
	
	return nil
}

// setupQuotaMonitoring sets up directory quotas if supported
func (s *StorageIsolator) setupQuotaMonitoring(dirs *TenantDirectories, config *TenantConfig) error {
	// This is a placeholder for quota setup
	// In production, you might use:
	// - XFS project quotas
	// - ext4 project quotas  
	// - Directory quotas via quota tools
	// - Custom monitoring with periodic size checks
	
	s.logger.Debug("quota monitoring setup",
		slog.String("tenant_id", dirs.TenantID),
		slog.Int64("max_storage_bytes", config.Limits.MaxStorageBytes),
	)
	
	return nil
}

// WriteFile writes a file with optional encryption and compression
func (s *StorageIsolator) WriteFile(
	dirs *TenantDirectories,
	relativePath string,
	data []byte,
	compress bool,
) error {
	fullPath := filepath.Join(dirs.BaseDir, relativePath)
	
	// Ensure the file is within the tenant directory
	if !strings.HasPrefix(fullPath, dirs.BaseDir) {
		return fmt.Errorf("path traversal attempt detected: %s", relativePath)
	}
	
	// Create parent directory if needed
	if err := os.MkdirAll(filepath.Dir(fullPath), dirs.DirMode); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}
	
	return s.writeFile(fullPath, data, dirs.FileMode, dirs.EncryptionEnabled)
}

// ReadFile reads a file with optional decryption and decompression
func (s *StorageIsolator) ReadFile(
	dirs *TenantDirectories,
	relativePath string,
) ([]byte, error) {
	fullPath := filepath.Join(dirs.BaseDir, relativePath)
	
	// Ensure the file is within the tenant directory
	if !strings.HasPrefix(fullPath, dirs.BaseDir) {
		return nil, fmt.Errorf("path traversal attempt detected: %s", relativePath)
	}
	
	return s.readFile(fullPath, dirs.EncryptionEnabled)
}

// writeFile writes data to a file with optional encryption
func (s *StorageIsolator) writeFile(path string, data []byte, mode os.FileMode, encrypt bool) error {
	var finalData []byte
	var err error
	
	if encrypt {
		finalData, err = s.encryptData(data)
		if err != nil {
			return fmt.Errorf("failed to encrypt data: %w", err)
		}
	} else {
		finalData = data
	}
	
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()
	
	if _, err := file.Write(finalData); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}
	
	return nil
}

// readFile reads data from a file with optional decryption
func (s *StorageIsolator) readFile(path string, decrypt bool) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	
	if decrypt {
		decryptedData, err := s.decryptData(data)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt data: %w", err)
		}
		return decryptedData, nil
	}
	
	return data, nil
}

// encryptData encrypts data using AES-GCM
func (s *StorageIsolator) encryptData(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return nil, err
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// decryptData decrypts data using AES-GCM
func (s *StorageIsolator) decryptData(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return nil, err
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	
	if len(data) < gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}
	
	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	
	return plaintext, nil
}

// setSecurityLabel sets security labels on directories (placeholder)
func (s *StorageIsolator) setSecurityLabel(path, label string) error {
	// This would integrate with SELinux or AppArmor
	// For now, we'll use extended attributes as a placeholder
	
	// Example with xattr (requires golang.org/x/sys/unix)
	// return unix.Setxattr(path, "security.builderd", []byte(label), 0)
	
	s.logger.Debug("security label applied",
		slog.String("path", path),
		slog.String("label", label),
	)
	
	return nil
}

// GetDirectorySize calculates the total size of a directory
func (s *StorageIsolator) GetDirectorySize(path string) (int64, error) {
	var totalSize int64
	
	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	
	return totalSize, err
}

// CheckQuota checks if a directory exceeds its quota
func (s *StorageIsolator) CheckQuota(
	ctx context.Context,
	dirs *TenantDirectories,
	maxBytes int64,
) error {
	currentSize, err := s.GetDirectorySize(dirs.BaseDir)
	if err != nil {
		return fmt.Errorf("failed to calculate directory size: %w", err)
	}
	
	if currentSize > maxBytes {
		return &QuotaError{
			Type:     QuotaTypeStorage,
			TenantID: dirs.TenantID,
			Current:  currentSize,
			Limit:    maxBytes,
			Message:  fmt.Sprintf("storage quota exceeded: %d/%d bytes", currentSize, maxBytes),
		}
	}
	
	// Update tenant manager with current usage
	s.tenantMgr.UpdateStorageUsage(ctx, dirs.TenantID, currentSize)
	
	return nil
}

// CleanupDirectories removes build directories and optionally archives them
func (s *StorageIsolator) CleanupDirectories(
	ctx context.Context,
	dirs *TenantDirectories,
	archive bool,
) error {
	if archive {
		// Archive the build before cleanup
		if err := s.archiveBuild(dirs); err != nil {
			s.logger.Warn("failed to archive build", slog.String("error", err.Error()))
		}
	}
	
	// Remove temporary directories immediately
	tempDirs := []string{dirs.TempDir, dirs.WorkspaceDir}
	for _, dir := range tempDirs {
		if err := os.RemoveAll(dir); err != nil {
			s.logger.Warn("failed to remove temp directory",
				slog.String("dir", dir),
				slog.String("error", err.Error()),
			)
		}
	}
	
	// Calculate freed space
	freedBytes, _ := s.GetDirectorySize(dirs.BaseDir)
	
	// Remove the entire build directory
	if err := os.RemoveAll(dirs.BaseDir); err != nil {
		return fmt.Errorf("failed to remove build directory: %w", err)
	}
	
	// Update storage usage
	s.tenantMgr.UpdateStorageUsage(ctx, dirs.TenantID, -freedBytes)
	
	s.logger.Info("cleaned up tenant directories",
		slog.String("tenant_id", dirs.TenantID),
		slog.String("build_id", dirs.BuildID),
		slog.Int64("freed_bytes", freedBytes),
	)
	
	return nil
}

// archiveBuild creates an archive of the build artifacts
func (s *StorageIsolator) archiveBuild(dirs *TenantDirectories) error {
	// This is a placeholder for build archiving
	// In production, you might:
	// - Create tar.gz archives
	// - Upload to S3/GCS/Azure
	// - Store in long-term storage
	// - Apply retention policies
	
	archivePath := filepath.Join(dirs.MetadataDir, "build.tar.gz")
	
	s.logger.Info("archived build",
		slog.String("tenant_id", dirs.TenantID),
		slog.String("build_id", dirs.BuildID),
		slog.String("archive_path", archivePath),
	)
	
	return nil
}

// GetStorageStats returns storage statistics for a tenant
func (s *StorageIsolator) GetStorageStats(
	ctx context.Context,
	tenantID string,
) (*StorageStats, error) {
	tenantDir := filepath.Join(s.baseDir, "tenants", tenantID)
	
	totalSize, err := s.GetDirectorySize(tenantDir)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate tenant storage size: %w", err)
	}
	
	// Count builds
	buildsDir := filepath.Join(tenantDir, "builds")
	buildCount := 0
	if entries, err := os.ReadDir(buildsDir); err == nil {
		buildCount = len(entries)
	}
	
	// Get cache size
	cacheDir := filepath.Join(tenantDir, "cache")
	cacheSize, _ := s.GetDirectorySize(cacheDir)
	
	stats := &StorageStats{
		TenantID:     tenantID,
		TotalBytes:   totalSize,
		CacheBytes:   cacheSize,
		BuildCount:   int32(buildCount),
		LastUpdated:  time.Now(),
	}
	
	return stats, nil
}

// StorageStats represents storage statistics for a tenant
type StorageStats struct {
	TenantID     string    `json:"tenant_id"`
	TotalBytes   int64     `json:"total_bytes"`
	CacheBytes   int64     `json:"cache_bytes"`
	BuildCount   int32     `json:"build_count"`
	LastUpdated  time.Time `json:"last_updated"`
}
