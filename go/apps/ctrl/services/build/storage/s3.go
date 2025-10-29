package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsS3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

type S3 struct {
	presigner *awsS3.PresignClient
	config    S3Config
	logger    logging.Logger
}

type S3Config struct {
	S3URL             string // Internal URL for S3 operations (e.g., http://s3:3902)
	S3PresignURL      string // Optional: External URL for presigned URLs when clients are outside Docker network. Defaults to S3URL.
	S3Bucket          string
	S3AccessKeyID     string
	S3AccessKeySecret string
	Logger            logging.Logger
}

func NewS3(config S3Config) (*S3, error) {
	logger := config.Logger.With("service", "storage")

	// Internal client config (for actual operations within Docker network)
	// This client is used for all actual S3 operations like CreateBucket, PutObject, GetObject
	//nolint: staticcheck
	internalResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...any) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               config.S3URL,
			HostnameImmutable: true,
		}, nil
	})

	internalCfg, err := awsConfig.LoadDefaultConfig(context.Background(),
		//nolint: staticcheck
		awsConfig.WithEndpointResolverWithOptions(internalResolver),
		awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			config.S3AccessKeyID,
			config.S3AccessKeySecret,
			"",
		)),
		awsConfig.WithRegion("auto"),
	)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to load aws config"))
	}

	// Create bucket using internal client
	internalClient := awsS3.NewFromConfig(internalCfg)
	logger.Info("creating bucket if necessary")
	_, err = internalClient.CreateBucket(context.Background(), &awsS3.CreateBucketInput{
		Bucket: aws.String(config.S3Bucket),
	})
	if err != nil && !strings.Contains(err.Error(), "BucketAlreadyOwnedByYou") {
		return nil, fmt.Errorf("failed to create bucket: %w", err)
	}

	// The reason for this check is when testing locally with docker CLI cannot access the internal S3 URL of minio without altering the /etc/host.
	// Thats why we conditionally check it and allow consumers of this package to decide,
	// but regardless of access URL we need to use internal S3 URL for bucket creation.
	presignURL := config.S3PresignURL
	if presignURL == "" {
		presignURL = config.S3URL // Default to internal
	}

	logger.Info("s3 storage initialized", "presign_url", presignURL)

	// Create presigner with the appropriate URL
	//nolint: staticcheck
	presignResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...any) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               presignURL,
			HostnameImmutable: true,
		}, nil
	})

	presignCfg, err := awsConfig.LoadDefaultConfig(context.Background(),
		//nolint: staticcheck
		awsConfig.WithEndpointResolverWithOptions(presignResolver),
		awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			config.S3AccessKeyID,
			config.S3AccessKeySecret,
			"",
		)),
		awsConfig.WithRegion("auto"),
	)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to load presign config"))
	}

	presignClient := awsS3.NewFromConfig(presignCfg)
	presigner := awsS3.NewPresignClient(presignClient)

	return &S3{
		presigner: presigner,
		config:    config,
		logger:    logger,
	}, nil
}

func (s *S3) GenerateDownloadURL(ctx context.Context, key string, expiresIn time.Duration) (string, error) {
	return s.presign(ctx, key, expiresIn, "GET")
}

func (s *S3) GenerateUploadURL(ctx context.Context, key string, expiresIn time.Duration) (string, error) {
	return s.presign(ctx, key, expiresIn, "PUT")
}

func (s *S3) presign(ctx context.Context, key string, expiresIn time.Duration, method string) (string, error) {
	logger := s.logger.With("method", method, "key", key, "expires_in", expiresIn.String())
	logger.Debug("presigning URL")

	opts := func(o *awsS3.PresignOptions) {
		o.Expires = expiresIn
	}

	var req *v4.PresignedHTTPRequest
	var err error

	switch method {
	case "PUT":
		req, err = s.presigner.PresignPutObject(ctx, &awsS3.PutObjectInput{
			Bucket: aws.String(s.config.S3Bucket),
			Key:    aws.String(key),
		}, opts)
	case "GET":
		req, err = s.presigner.PresignGetObject(ctx, &awsS3.GetObjectInput{
			Bucket: aws.String(s.config.S3Bucket),
			Key:    aws.String(key),
		}, opts)
	default:
		logger.Error("unsupported HTTP method", "error", fmt.Sprintf("method %s is not supported", method))
		return "", fmt.Errorf("unsupported method: %s", method)
	}

	if err != nil {
		logger.Error("failed to presign URL", "error", err)
		return "", fmt.Errorf("failed to presign %s: %w", method, err)
	}

	return req.URL, nil
}
