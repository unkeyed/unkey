package containers

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsS3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
)

const (
	s3Port = 3902

	s3AccessKey = "garage-test-access-key"
	s3SecretKey = "garage-test-secret-key"
)

// S3Config holds connection information for the S3-compatible test container.
//
// The returned configuration can be used directly with AWS SDK, MinIO client,
// or any S3-compatible client library.
type S3Config struct {
	// URL is the S3 endpoint URL (e.g., "http://localhost:54321").
	URL string

	// AccessKeyID is the access key for authentication.
	AccessKeyID string

	// SecretAccessKey is the secret key for authentication.
	SecretAccessKey string
}

// S3 starts the shared Docker Compose Garage service with a pre-created vault bucket.
// Use [S3Config.CreateBucket] when a test needs an isolated bucket.
func S3(t testing.TB) S3Config {
	t.Helper()

	c := startService(t, "s3")

	return S3Config{
		URL:             fmt.Sprintf("http://%s", c.Addr(t, s3Port)),
		AccessKeyID:     s3AccessKey,
		SecretAccessKey: s3SecretKey,
	}
}

// CreateBucket provisions a uniquely named bucket for test isolation.
func (c S3Config) CreateBucket(t testing.TB) string {
	t.Helper()

	client := awsS3.New(awsS3.Options{
		BaseEndpoint: aws.String(c.URL),
		Region:       "auto",
		Credentials:  credentials.NewStaticCredentialsProvider(c.AccessKeyID, c.SecretAccessKey, ""),
		UsePathStyle: true,
	})
	bucket := "test-" + uid.DNS1035(20)
	_, err := client.CreateBucket(t.Context(), &awsS3.CreateBucketInput{
		Bucket: aws.String(bucket),
	})
	require.NoError(t, err)
	return bucket
}
