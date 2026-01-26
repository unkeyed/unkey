// Package s3 provides pre-signed URL generation for S3-compatible object storage.
//
// The package supports separate internal and external S3 endpoints, which is
// necessary when the service runs inside a Docker network but clients access
// storage from outside. For example, the service may communicate with MinIO at
// http://minio:9000 internally, while clients need URLs pointing to
// http://localhost:9000.
//
// # Key Types
//
// [S3] is the main client that implements the [Storage] interface. Create one
// with [NewS3] using [S3Config] for configuration.
//
// # Usage
//
//	s3Client, err := s3.NewS3(s3.S3Config{
//		Logger:            logger,
//		S3URL:             "http://minio:9000",
//		S3PresignURL:      "http://localhost:9000",
//		S3Bucket:          "artifacts",
//		S3AccessKeyID:     "access-key",
//		S3AccessKeySecret: "secret-key",
//	})
//	if err != nil {
//		// Handle error - bucket creation or AWS config failed
//	}
//
//	// Generate a URL for clients to upload an artifact
//	uploadURL, err := s3Client.GenerateUploadURL(ctx, "builds/123/artifact.tar.gz", time.Hour)
//
//	// Generate a URL for clients to download an artifact
//	downloadURL, err := s3Client.GenerateDownloadURL(ctx, "builds/123/artifact.tar.gz", time.Hour)
package s3
