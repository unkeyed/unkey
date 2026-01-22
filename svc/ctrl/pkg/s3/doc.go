// Package storage provides S3-compatible storage for build artifacts.
//
// This package implements S3-compatible object storage for
// container build artifacts. It supports multiple S3 endpoints
// and provides pre-signed URL generation for secure artifact access.
//
// # Architecture
//
// The storage package provides:
//   - S3 client abstraction for multiple providers
//   - Pre-signed URL generation for secure artifact downloads
//   - Upload functionality for build artifact storage
//   - Integration with S3-compatible storage backends
//
// # Key Features
//
// - Multiple S3 provider support (AWS, MinIO, localstack, etc.)
//   - Secure pre-signed URL generation with configurable TTL
//   - Multipart upload support for large artifacts
//   - Error handling with retry logic
//   - Logging for storage operations debugging
//
// # Usage
//
// Creating S3 storage:
//
//	storage, err := storage.NewS3(storage.S3Config{
//		Logger:            logger,
//		S3URL:             "https://s3.amazonaws.com",
//		S3Bucket:          "build-artifacts",
//		S3AccessKeyID:     "access-key",
//		S3AccessKeySecret: "secret-key",
//	})
//
//	// Upload build artifact
//	err = storage.Upload(ctx, "build-artifact.tar.gz", buildArtifactData)
//
//	// Generate download URL
//	url, err := storage.GeneratePresignedURL(ctx, "build-artifact.tar.gz", time.Hour)
//
// # Error Handling
//
// The package provides comprehensive error handling for S3 operations
// including network failures, permission errors, and invalid
// configurations.
package s3
