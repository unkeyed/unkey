package dockertest

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsS3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/require"
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

	// Bucket is the name of the ephemeral bucket created for this test.
	// The bucket is automatically deleted when the test completes.
	Bucket string

	// AccessKeyID is the access key for authentication.
	AccessKeyID string

	// SecretAccessKey is the secret key for authentication.
	SecretAccessKey string
}

var s3Ctr shared

// s3ContainerConfig returns the container configuration for MinIO/S3.
func s3ContainerConfig() containerConfig {
	return containerConfig{
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
		SkipCleanup:  false,
	}
}

// S3 starts (or reuses) a shared MinIO container and returns an ephemeral bucket
// for this test. The bucket is deleted when the test completes.
//
// The container starts on the first call in the process and is reused by all
// subsequent calls. Each call creates a unique bucket, ensuring complete
// isolation between tests.
func S3(t *testing.T) S3Config {
	t.Helper()

	ctr := s3Ctr.get(t, s3ContainerConfig())

	port := ctr.Port(minioPort)
	url := fmt.Sprintf("http://localhost:%s", port)

	// Create an ephemeral bucket for this test.
	bucketName := fmt.Sprintf("test-%d", time.Now().UnixNano())

	client := newS3ClientInternal(t, url, minioAccessKey, minioSecretKey)

	ctx := context.Background()
	_, err := client.CreateBucket(ctx, &awsS3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})
	require.NoError(t, err)

	// Clean up: delete the bucket when the test finishes.
	t.Cleanup(func() {
		cleanupCtx := context.Background()
		// List and delete all objects first (bucket must be empty to delete).
		listOutput, listErr := client.ListObjectsV2(cleanupCtx, &awsS3.ListObjectsV2Input{
			Bucket: aws.String(bucketName),
		})
		if listErr == nil {
			for _, obj := range listOutput.Contents {
				_, _ = client.DeleteObject(cleanupCtx, &awsS3.DeleteObjectInput{
					Bucket: aws.String(bucketName),
					Key:    obj.Key,
				})
			}
		}
		_, deleteErr := client.DeleteBucket(cleanupCtx, &awsS3.DeleteBucketInput{
			Bucket: aws.String(bucketName),
		})
		if deleteErr != nil {
			t.Logf("  S3 cleanup: failed to delete bucket %s: %v", bucketName, deleteErr)
		}
	})

	return S3Config{
		URL:            url,
		Bucket:         bucketName,
		AccessKeyID:    minioAccessKey,
		SecretAccessKey: minioSecretKey,
	}
}

// newS3ClientInternal creates an S3 client for internal use (bucket creation/cleanup).
func newS3ClientInternal(t *testing.T, url, accessKey, secretKey string) *awsS3.Client {
	t.Helper()

	//nolint:staticcheck
	resolver := aws.EndpointResolverWithOptionsFunc(
		func(service, region string, options ...any) (aws.Endpoint, error) {
			//nolint:staticcheck
			return aws.Endpoint{
				URL:               url,
				HostnameImmutable: true,
			}, nil
		},
	)

	cfg, err := awsConfig.LoadDefaultConfig(context.Background(),
		awsConfig.WithEndpointResolverWithOptions(resolver), //nolint:staticcheck
		awsConfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		),
		awsConfig.WithRegion("us-east-1"),
	)
	require.NoError(t, err)

	return awsS3.NewFromConfig(cfg)
}
