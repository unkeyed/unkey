package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsS3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"

	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/vault/internal/metrics"
)

type s3 struct {
	client *awsS3.Client
	config S3Config
}

type S3Config struct {
	S3URL             string
	S3Bucket          string
	S3AccessKeyID     string
	S3AccessKeySecret string
}

func NewS3(config S3Config) (Storage, error) {
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

	logger.Info("s3 storage initialized")

	return &s3{config: config, client: client}, nil
}

func (s *s3) Key(workspaceId string, dekID string) string {
	return fmt.Sprintf("%s/%s", workspaceId, dekID)
}

func (s *s3) Latest(workspaceId string) string {
	return s.Key(workspaceId, "LATEST")
}

func (s *s3) PutObject(ctx context.Context, key string, data []byte) (err error) {
	defer func() { observeS3("put", err) }()
	_, err = s.client.PutObject(ctx, &awsS3.PutObjectInput{
		Bucket: aws.String(s.config.S3Bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return fmt.Errorf("failed to put object: %w", err)
	}
	return nil
}

func (s *s3) GetObject(ctx context.Context, key string) (data []byte, found bool, err error) {
	defer func() { observeS3("get", err) }()
	o, err := s.client.GetObject(ctx, &awsS3.GetObjectInput{
		Bucket: aws.String(s.config.S3Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		// Bare 404s are compatible misses, but explicit errors such as
		// NoSuchBucket must not trigger creation of replacement keys.
		code := ""
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			code = apiErr.ErrorCode()
		}
		var respErr *awshttp.ResponseError
		if code == "NoSuchKey" || ((code == "" || code == "NotFound") && errors.As(err, &respErr) && respErr.HTTPStatusCode() == http.StatusNotFound) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("failed to get object: %w", err)
	}
	defer func() { _ = o.Body.Close() }()
	b, err := io.ReadAll(o.Body)
	if err != nil {
		return nil, false, fmt.Errorf("failed to read object: %w", err)
	}
	return b, true, nil
}

func (s *s3) ListObjectKeys(ctx context.Context, prefix string) (keys []string, err error) {
	defer func() { observeS3("list", err) }()
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
	keys = make([]string, len(o.Contents))
	for i, obj := range o.Contents {
		keys[i] = *obj.Key
	}
	return keys, nil
}

func observeS3(operation string, err error) {
	outcome, code := "success", ""
	if errors.Is(err, context.Canceled) {
		outcome = "canceled"
	} else if err != nil {
		outcome, code = "error", "other"
		if errors.Is(err, context.DeadlineExceeded) {
			code = "timeout"
		}
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			// Provider messages and unknown codes can contain secrets or object keys.
			switch apiErr.ErrorCode() {
			case "InvalidAccessKeyId", "SignatureDoesNotMatch", "AccessDenied", "ExpiredToken", "NoSuchBucket":
				code = apiErr.ErrorCode()
			}
		}
		status := 0
		var respErr *awshttp.ResponseError
		if errors.As(err, &respErr) {
			status = respErr.HTTPStatusCode()
		}
		logger.Error("vault s3 operation failed", "operation", operation, "error_code", code, "http_status", status)
	}
	metrics.S3OperationsTotal.WithLabelValues(operation, outcome, code).Inc()
}
