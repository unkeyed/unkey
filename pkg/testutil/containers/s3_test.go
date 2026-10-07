package containers_test

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsS3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
)

// TestS3 verifies that container startup provisions a bucket usable through S3.
func TestS3(t *testing.T) {
	s3Cfg := containers.S3(t)

	client := newS3Client(t, s3Cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bucketName := "vault"

	// Put an object
	testKey := "test-key"
	testData := []byte("hello, world!")
	_, err := client.PutObject(ctx, &awsS3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(testKey),
		Body:   bytes.NewReader(testData),
	})
	require.NoError(t, err)

	// Get the object and verify contents
	resp, err := client.GetObject(ctx, &awsS3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(testKey),
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })

	retrievedData, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, testData, retrievedData)
}

func TestS3_ReusesContainer(t *testing.T) {
	s3Cfg1 := containers.S3(t)
	s3Cfg2 := containers.S3(t)

	require.Equal(t, s3Cfg1.URL, s3Cfg2.URL)

	client1 := newS3Client(t, s3Cfg1)
	client2 := newS3Client(t, s3Cfg2)

	ctx := context.Background()

	bucketName := s3Cfg1.CreateBucket(t)

	testKey := "test-key"
	_, err := client1.PutObject(ctx, &awsS3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(testKey),
		Body:   bytes.NewReader([]byte("shared data")),
	})
	require.NoError(t, err)

	resp2, err := client2.GetObject(ctx, &awsS3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(testKey),
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, resp2.Body.Close()) })
	data2, err := io.ReadAll(resp2.Body)
	require.NoError(t, err)
	require.Equal(t, []byte("shared data"), data2)
}

// newS3Client creates an S3 client configured for the Garage test container.
func newS3Client(t *testing.T, s3Cfg containers.S3Config) *awsS3.Client {
	t.Helper()

	// nolint:staticcheck
	resolver := aws.EndpointResolverWithOptionsFunc(
		func(service, region string, options ...any) (aws.Endpoint, error) {
			// nolint:staticcheck
			return aws.Endpoint{
				URL:               s3Cfg.URL,
				HostnameImmutable: true,
			}, nil
		},
	)

	cfg, err := awsConfig.LoadDefaultConfig(context.Background(),
		awsConfig.WithEndpointResolverWithOptions(resolver), // nolint:staticcheck
		awsConfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(s3Cfg.AccessKeyID, s3Cfg.SecretAccessKey, ""),
		),
		awsConfig.WithRegion("auto"),
	)
	require.NoError(t, err)

	return awsS3.NewFromConfig(cfg)
}
