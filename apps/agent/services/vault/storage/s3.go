package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsS3 "github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
)

type s3 struct {
	client *awsS3.Client
	config S3Config
	logger logging.Logger
}

type S3Config struct {
	S3URL             string
	S3Bucket          string
	S3AccessKeyId     string
	S3AccessKeySecret string
	Logger            logging.Logger
}

func NewS3(config S3Config) (Storage, error) {

	logger := config.Logger.With().Str("service", "storage").Logger()

	logger.Info().Msg("using s3 storage")

	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...any) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               config.S3URL,
			HostnameImmutable: true,
		}, nil

	})

	cfg, err := awsConfig.LoadDefaultConfig(context.Background(),
		awsConfig.WithEndpointResolverWithOptions(r2Resolver),
		awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(config.S3AccessKeyId, config.S3AccessKeySecret, "")),
		awsConfig.WithRegion("auto"),
		awsConfig.WithRetryMode(aws.RetryModeStandard),
		awsConfig.WithRetryMaxAttempts(3),
	)

	if err != nil {
		return nil, fault.Wrap(err, fmsg.With("failed to load aws config"))
	}

	client := awsS3.NewFromConfig(cfg)
	logger.Info().Msg("creating bucket if necessary")
	_, err = client.CreateBucket(context.Background(), &awsS3.CreateBucketInput{
		Bucket: aws.String(config.S3Bucket),
	})
	if err != nil && !strings.Contains(err.Error(), "BucketAlreadyOwnedByYou") {
		return nil, fault.Wrap(err, fmsg.With("failed to create bucket"))
	}

	logger.Info().Msg("s3 storage initialized")

	return &s3{config: config, client: client, logger: logger}, nil
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
		return nil, fault.Wrap(err, fmsg.With("failed to list objects"))
	}
	keys := make([]string, len(o.Contents))
	for i, obj := range o.Contents {
		keys[i] = *obj.Key
	}
	return keys, nil
}
