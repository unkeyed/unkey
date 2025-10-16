package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsS3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

type S3 struct {
	client                *awsS3.Client
	presignClient         *awsS3.PresignClient
	externalPresignClient *awsS3.PresignClient // For generating URLs accessible from outside Docker network
	config                S3Config
	logger                logging.Logger
}

type S3Config struct {
	S3URL             string // Internal Docker network URL (e.g., http://s3:3902)
	S3ExternalURL     string // Host-accessible URL (e.g., http://localhost:3902)
	S3Bucket          string
	S3AccessKeyID     string
	S3AccessKeySecret string
	Logger            logging.Logger
}

func NewS3(config S3Config) (*S3, error) {
	logger := config.Logger.With("service", "storage")
	logger.Info("using s3 storage", "internal_url", config.S3URL, "external_url", config.S3ExternalURL)

	// Internal client config (for actual operations within Docker network)
	// This client is used for all actual S3 operations like CreateBucket, PutObject, GetObject
	internalResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...any) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               config.S3URL,
			HostnameImmutable: true,
		}, nil
	})

	internalCfg, err := awsConfig.LoadDefaultConfig(context.Background(),
		awsConfig.WithEndpointResolverWithOptions(internalResolver),
		awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(config.S3AccessKeyID, config.S3AccessKeySecret, "")),
		awsConfig.WithRegion("auto"),
		awsConfig.WithRetryMode(aws.RetryModeStandard),
		awsConfig.WithRetryMaxAttempts(3),
	)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to load aws config"), fault.Public("failed to load aws config"))
	}

	client := awsS3.NewFromConfig(internalCfg, func(o *awsS3.Options) {
		o.UsePathStyle = true
	})
	presignClient := awsS3.NewPresignClient(client)

	// Create bucket using internal client ONLY
	// Bucket operations must use the internal endpoint since we're running inside Docker
	logger.Info("creating bucket if necessary")
	_, err = client.CreateBucket(context.Background(), &awsS3.CreateBucketInput{
		Bucket: aws.String(config.S3Bucket),
	})
	if err != nil && !strings.Contains(err.Error(), "BucketAlreadyOwnedByYou") {
		return nil, fmt.Errorf("failed to create bucket: %w", err)
	}

	// External presign client (ONLY for generating presigned URLs accessible from host machine)
	// S3 presigned URLs include the hostname in the cryptographic signature, so we cannot simply
	// string-replace hostnames after generation. Instead, we create a separate client configured
	// with the external endpoint to generate URLs that will be valid when accessed from outside
	// the Docker network (e.g., by the CLI running on the host machine).
	var externalPresignClient *awsS3.PresignClient
	if config.S3ExternalURL != "" && config.S3ExternalURL != config.S3URL {
		externalResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...any) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:               config.S3ExternalURL,
				HostnameImmutable: true,
			}, nil
		})

		externalCfg, err := awsConfig.LoadDefaultConfig(context.Background(),
			awsConfig.WithEndpointResolverWithOptions(externalResolver),
			awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(config.S3AccessKeyID, config.S3AccessKeySecret, "")),
			awsConfig.WithRegion("auto"),
		)
		if err != nil {
			return nil, fault.Wrap(err, fault.Internal("failed to load external aws config"), fault.Public("failed to load external aws config"))
		}

		externalClient := awsS3.NewFromConfig(externalCfg, func(o *awsS3.Options) {
			o.UsePathStyle = true
		})
		externalPresignClient = awsS3.NewPresignClient(externalClient)
		logger.Info("external presign client configured for CLI access")
	}

	logger.Info("s3 storage initialized")
	return &S3{
		config:                config,
		client:                client,
		presignClient:         presignClient,
		externalPresignClient: externalPresignClient,
		logger:                logger,
	}, nil
}

// GetPresignedURL generates a presigned URL for downloading an object using the internal endpoint
func (s *S3) GetPresignedURL(ctx context.Context, key string, expiresIn time.Duration) (string, error) {
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

// PutPresignedURL generates a presigned URL for uploading an object using the internal endpoint
func (s *S3) PutPresignedURL(ctx context.Context, key string, expiresIn time.Duration) (string, error) {
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

// GetPresignedURLExternal generates a presigned URL for downloading using the external endpoint
// Use this when generating URLs for clients outside the Docker network (e.g., CLI on host machine)
func (s *S3) GetPresignedURLExternal(ctx context.Context, key string, expiresIn time.Duration) (string, error) {
	presigner := s.presignClient
	if s.externalPresignClient != nil {
		presigner = s.externalPresignClient
	}

	req, err := presigner.PresignGetObject(ctx, &awsS3.GetObjectInput{
		Bucket: aws.String(s.config.S3Bucket),
		Key:    aws.String(key),
	}, func(opts *awsS3.PresignOptions) {
		opts.Expires = expiresIn
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate external presigned GET URL: %w", err)
	}
	return req.URL, nil
}

// PutPresignedURLExternal generates a presigned URL for uploading using the external endpoint
// Use this when generating URLs for clients outside the Docker network (e.g., CLI on host machine)
func (s *S3) PutPresignedURLExternal(ctx context.Context, key string, expiresIn time.Duration) (string, error) {
	presigner := s.presignClient
	if s.externalPresignClient != nil {
		presigner = s.externalPresignClient
	}

	req, err := presigner.PresignPutObject(ctx, &awsS3.PutObjectInput{
		Bucket: aws.String(s.config.S3Bucket),
		Key:    aws.String(key),
	}, func(opts *awsS3.PresignOptions) {
		opts.Expires = expiresIn
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate external presigned PUT URL: %w", err)
	}
	return req.URL, nil
}
