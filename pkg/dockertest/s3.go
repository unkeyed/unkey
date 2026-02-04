package dockertest

import (
	"fmt"
	"testing"
	"time"
)

const (
	minioImage = "quay.io/minio/minio:latest"
	minioPort  = "9000/tcp"

	// Default MinIO credentials used for test containers.
	minioAccessKey = "minioadmin"
	minioSecretKey = "minioadmin"
)

// S3Config holds connection information for an S3-compatible container.
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

// S3 starts a MinIO container and returns the connection configuration.
//
// MinIO is an S3-compatible object storage server. The container is configured
// with default credentials (minioadmin/minioadmin) and a single server instance
// suitable for testing.
//
// The container is automatically removed when the test completes via t.Cleanup.
// This function blocks until MinIO's health endpoint responds (up to 30s).
// Fails the test if Docker is unavailable or the container fails to start.
//
// Example usage:
//
//	func TestS3Integration(t *testing.T) {
//	    s3 := dockertest.S3(t)
//	    client, err := minio.New(s3.URL, &minio.Options{
//	        Creds: credentials.NewStaticV4(s3.AccessKeyID, s3.SecretAccessKey, ""),
//	    })
//	    require.NoError(t, err)
//	    // Use client...
//	}
func S3(t *testing.T) S3Config {
	t.Helper()

	ctr := startContainer(t, containerConfig{
		Image:        minioImage,
		ExposedPorts: []string{minioPort},
		Env: map[string]string{
			"MINIO_ROOT_USER":     minioAccessKey,
			"MINIO_ROOT_PASSWORD": minioSecretKey,
		},
		Cmd:          []string{"server", "/data"},
		WaitStrategy: NewHTTPWait(minioPort, "/minio/health/live"),
		WaitTimeout:  30 * time.Second,
		Tmpfs:        nil,
	})

	port := ctr.Port(minioPort)
	return S3Config{
		URL:             fmt.Sprintf("http://localhost:%s", port),
		AccessKeyID:     minioAccessKey,
		SecretAccessKey: minioSecretKey,
	}
}
