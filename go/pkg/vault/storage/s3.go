package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsS3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

type s3 struct {
	client        *awsS3.Client
	presignClient *awsS3.PresignClient
	config        S3Config
	logger        logging.Logger
}

type S3Config struct {
	S3URL             string
	S3Bucket          string
	S3AccessKeyID     string
	S3AccessKeySecret string
	Logger            logging.Logger
}

func NewS3(config S3Config) (Storage, error) {
	logger := config.Logger.With("service", "storage")
	logger.Info("using s3 storage")

	// nolint:staticcheck
	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...any) (aws.Endpoint, error) {
		// nolint:staticcheck
		return aws.Endpoint{
			URL:               config.S3URL,
			HostnameImmutable: true,
		}, nil
	})

	cfg, err := awsConfig.LoadDefaultConfig(context.Background(),
		awsConfig.WithEndpointResolverWithOptions(r2Resolver), // nolint:staticcheck
		awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(config.S3AccessKeyID, config.S3AccessKeySecret, "")),
		awsConfig.WithRegion("auto"),
		awsConfig.WithRetryMode(aws.RetryModeStandard),
		awsConfig.WithRetryMaxAttempts(3),
	)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to load aws config"), fault.Public("failed to load aws config"))
	}

	client := awsS3.NewFromConfig(cfg)
	presignClient := awsS3.NewPresignClient(client)

	logger.Info("creating bucket if necessary")
	_, err = client.CreateBucket(context.Background(), &awsS3.CreateBucketInput{
		Bucket: aws.String(config.S3Bucket),
	})
	if err != nil && !strings.Contains(err.Error(), "BucketAlreadyOwnedByYou") {
		return nil, fmt.Errorf("failed to create bucket: %w", err)
	}

	logger.Info("s3 storage initialized")
	return &s3{
		config:        config,
		client:        client,
		presignClient: presignClient,
		logger:        logger,
	}, nil
}

func (s *s3) Key(workspaceId string, dekID string) string {
	return fmt.Sprintf("%s/%s", workspaceId, dekID)
}

func (s *s3) Latest(workspaceId string) string {
	return s.Key(workspaceId, "LATEST")
}

func (s *s3) PutObject(ctx context.Context, key string, data []byte) error {
	_, err := s.client.PutObject(ctx, &awsS3.PutObjectInput{
		Bucket: aws.String(s.config.S3Bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return fmt.Errorf("failed to put object: %w", err)
	}
	return nil
}

func (s *s3) GetObject(ctx context.Context, key string) ([]byte, bool, error) {
	o, err := s.client.GetObject(ctx, &awsS3.GetObjectInput{
		Bucket: aws.String(s.config.S3Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if strings.Contains(err.Error(), "StatusCode: 404") {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("failed to get object: %w", err)
	}
	defer o.Body.Close()

	b, err := io.ReadAll(o.Body)
	if err != nil {
		return nil, false, fmt.Errorf("failed to read object: %w", err)
	}

	return b, true, nil
}

func (s *s3) ListObjectKeys(ctx context.Context, prefix string) ([]string, error) {
	input := &awsS3.ListObjectsV2Input{
		Bucket: aws.String(s.config.S3Bucket),
	}
	if prefix != "" {
		input.Prefix = aws.String(prefix)
	}

	o, err := s.client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	keys := make([]string, len(o.Contents))
	for i, obj := range o.Contents {
		keys[i] = *obj.Key
	}

	return keys, nil
}

// GetPresignedURL generates a presigned URL for downloading an object
func (s *s3) GetPresignedURL(ctx context.Context, key string, expiresIn time.Duration) (string, error) {
	req, err := s.presignClient.PresignGetObject(ctx, &awsS3.GetObjectInput{
		Bucket: aws.String(s.config.S3Bucket),
		Key:    aws.String(key),
	}, func(opts *awsS3.PresignOptions) {
		opts.Expires = expiresIn
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return req.URL, nil
}

// PutPresignedURL generates a presigned URL for uploading an object
func (s *s3) PutPresignedURL(ctx context.Context, key string, expiresIn time.Duration) (string, error) {
	req, err := s.presignClient.PresignPutObject(ctx, &awsS3.PutObjectInput{
		Bucket: aws.String(s.config.S3Bucket),
		Key:    aws.String(key),
	}, func(opts *awsS3.PresignOptions) {
		opts.Expires = expiresIn
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return req.URL, nil
}
