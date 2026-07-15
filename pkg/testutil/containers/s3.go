package containers

import (
	"fmt"
	"testing"
)

const (
	minioPort = 9000

	// Default MinIO credentials used for the test container.
	minioAccessKey = "minioadmin"
	minioSecretKey = "minioadmin"
)

// S3Config holds connection information for the S3-compatible test container.
//
// The returned configuration can be used directly with AWS SDK, MinIO client,
// or any S3-compatible client library. Credentials are set to MinIO defaults.
type S3Config struct {
	// URL is the S3 endpoint URL (e.g., "http://localhost:54321").
	URL string

	// AccessKeyID is the access key for authentication.
	AccessKeyID string

	// SecretAccessKey is the secret key for authentication.
	SecretAccessKey string
}

// S3 starts the shared Docker Compose MinIO service and returns connection info.
//
// MinIO is an S3-compatible object storage server. The container is configured
// with default credentials (minioadmin/minioadmin) and a single server instance
// suitable for testing.
//
// The container is reused through the worktree's Docker Compose project.
//
// Example usage:
//
//	func TestS3Integration(t *testing.T) {
//	    s3 := containers.S3(t)
//	    client, err := minio.New(s3.URL, &minio.Options{
//	        Creds: credentials.NewStaticV4(s3.AccessKeyID, s3.SecretAccessKey, ""),
//	    })
//	    require.NoError(t, err)
//	    // Use client...
//	}
func S3(t testing.TB) S3Config {
	t.Helper()

	c := startService(t, "s3")

	return S3Config{
		URL:             fmt.Sprintf("http://%s", c.Addr(t, minioPort)),
		AccessKeyID:     minioAccessKey,
		SecretAccessKey: minioSecretKey,
	}
}
