package dockertest_test

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
	"github.com/unkeyed/unkey/pkg/dockertest"
)

// TestS3 verifies that the MinIO container starts correctly and is accessible
// via the AWS S3 SDK.
func TestS3(t *testing.T) {
	s3Cfg := dockertest.S3(t)

	client := newS3Client(t, s3Cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create a test bucket
	bucketName := "test-bucket"
	_, err := client.CreateBucket(ctx, &awsS3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})
	require.NoError(t, err)

	// Put an object
	testKey := "test-key"
	testData := []byte("hello, world!")
	_, err = client.PutObject(ctx, &awsS3.PutObjectInput{
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
	defer resp.Body.Close()

	retrievedData, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, testData, retrievedData)
}

// TestS3_MultipleContainers verifies that multiple MinIO containers can run
// in parallel with isolated data.
func TestS3_MultipleContainers(t *testing.T) {
	s3Cfg1 := dockertest.S3(t)
	s3Cfg2 := dockertest.S3(t)

	// The URLs should be different (different ports)
	require.NotEqual(t, s3Cfg1.URL, s3Cfg2.URL)

	client1 := newS3Client(t, s3Cfg1)
	client2 := newS3Client(t, s3Cfg2)

	ctx := context.Background()

	// Create bucket with same name in both containers
	bucketName := "shared-bucket-name"
	_, err := client1.CreateBucket(ctx, &awsS3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})
	require.NoError(t, err)

	_, err = client2.CreateBucket(ctx, &awsS3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})
	require.NoError(t, err)

	// Put different data with same key in each container
	testKey := "test-key"
	_, err = client1.PutObject(ctx, &awsS3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(testKey),
		Body:   bytes.NewReader([]byte("data from container 1")),
	})
	require.NoError(t, err)

	_, err = client2.PutObject(ctx, &awsS3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(testKey),
		Body:   bytes.NewReader([]byte("data from container 2")),
	})
	require.NoError(t, err)

	// Verify isolation - each container has its own data
	resp1, err := client1.GetObject(ctx, &awsS3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(testKey),
	})
	require.NoError(t, err)
	defer resp1.Body.Close()
	data1, err := io.ReadAll(resp1.Body)
	require.NoError(t, err)
	require.Equal(t, []byte("data from container 1"), data1)

	resp2, err := client2.GetObject(ctx, &awsS3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(testKey),
	})
	require.NoError(t, err)
	defer resp2.Body.Close()
	data2, err := io.ReadAll(resp2.Body)
	require.NoError(t, err)
	require.Equal(t, []byte("data from container 2"), data2)
}

// newS3Client creates an S3 client configured for the given MinIO container.
func newS3Client(t *testing.T, s3Cfg dockertest.S3Config) *awsS3.Client {
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
		awsConfig.WithRegion("us-east-1"),
	)
	require.NoError(t, err)

	return awsS3.NewFromConfig(cfg)
}
