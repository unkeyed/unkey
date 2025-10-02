package iceberg

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/catalog"
	_ "github.com/apache/iceberg-go/catalog/rest" // Register REST catalog
	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	parquet "github.com/segmentio/parquet-go"
	"github.com/unkeyed/unkey/go/pkg/analytics"
	"github.com/unkeyed/unkey/go/pkg/batch"
	"github.com/unkeyed/unkey/go/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// Parquet-compatible structs with parquet tags
type KeyVerificationParquet struct {
	RequestID    string   `parquet:"request_id,snappy"`
	Time         int64    `parquet:"time,snappy"`
	WorkspaceID  string   `parquet:"workspace_id,snappy"`
	KeySpaceID   string   `parquet:"key_space_id,snappy"`
	IdentityID   string   `parquet:"identity_id,snappy"`
	KeyID        string   `parquet:"key_id,snappy"`
	Region       string   `parquet:"region,snappy"`
	Outcome      string   `parquet:"outcome,snappy"`
	Tags         []string `parquet:"tags,list,snappy"`
	SpentCredits int64    `parquet:"spent_credits,snappy"`
	Latency      float64  `parquet:"latency,snappy"`
}

type RatelimitParquet struct {
	RequestID   string  `parquet:"request_id,snappy"`
	Time        int64   `parquet:"time,snappy"`
	WorkspaceID string  `parquet:"workspace_id,snappy"`
	NamespaceID string  `parquet:"namespace_id,snappy"`
	Identifier  string  `parquet:"identifier,snappy"`
	Passed      bool    `parquet:"passed,snappy"`
	Latency     float64 `parquet:"latency,snappy"`
	OverrideID  string  `parquet:"override_id,snappy"`
	Limit       uint64  `parquet:"limit,snappy"`
	Remaining   uint64  `parquet:"remaining,snappy"`
	ResetAt     int64   `parquet:"reset_at,snappy"`
}

type ApiRequestParquet struct {
	RequestID       string   `parquet:"request_id,snappy"`
	Time            int64    `parquet:"time,snappy"`
	WorkspaceID     string   `parquet:"workspace_id,snappy"`
	Host            string   `parquet:"host,snappy"`
	Method          string   `parquet:"method,snappy"`
	Path            string   `parquet:"path,snappy"`
	QueryString     string   `parquet:"query_string,snappy"`
	QueryParams     string   `parquet:"query_params,snappy"` // JSON string
	RequestHeaders  []string `parquet:"request_headers,list,snappy"`
	RequestBody     string   `parquet:"request_body,snappy"`
	ResponseStatus  int32    `parquet:"response_status,snappy"`
	ResponseHeaders []string `parquet:"response_headers,list,snappy"`
	ResponseBody    string   `parquet:"response_body,snappy"`
	Error           string   `parquet:"error,snappy"`
	ServiceLatency  int64    `parquet:"service_latency,snappy"`
	UserAgent       string   `parquet:"user_agent,snappy"`
	IpAddress       string   `parquet:"ip_address,snappy"`
	Region          string   `parquet:"region,snappy"`
}

// Writer implements the analytics.Writer interface for customer-specific data lakes.
// Each customer gets their own isolated data lake (e.g., R2 bucket with Iceberg format).
// This writer:
// 1. Batches events in memory (1000 events or 30 seconds) using batch.BatchProcessor
// 2. Writes batches to Parquet files using segmentio/parquet-go
// 3. Uploads Parquet files to S3-compatible storage
// 4. Commits files to Iceberg table via REST Catalog API
// 5. Uses circuit breakers per workspace to fail fast after consecutive errors
type Writer struct {
	config        Config
	logger        logging.Logger
	s3Client      *s3.Client
	icebergClient catalog.Catalog

	// Batch processors for each event type
	keyVerificationProcessor *batch.BatchProcessor[KeyVerificationParquet]
	ratelimitProcessor       *batch.BatchProcessor[RatelimitParquet]
	apiRequestProcessor      *batch.BatchProcessor[ApiRequestParquet]

	// Circuit breakers per workspace to fail fast after consecutive errors
	circuitBreakers   map[string]circuitbreaker.CircuitBreaker[struct{}]
	circuitBreakersMu sync.RWMutex
}

// New creates a new data lake writer with the provided configuration.
func New(config Config, logger logging.Logger) (analytics.Writer, error) {
	if config.Bucket == "" {
		return nil, fault.New("bucket is required")
	}

	// Initialize S3-compatible client for writing Parquet files
	// Use the deprecated EndpointResolverWithOptions for R2 compatibility
	// nolint:staticcheck
	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...any) (aws.Endpoint, error) {
		// nolint:staticcheck
		if config.Endpoint != "" {
			return aws.Endpoint{
				URL:               config.Endpoint,
				HostnameImmutable: true,
			}, nil
		}
		// Use default resolver if no endpoint specified
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	cfg, err := awsConfig.LoadDefaultConfig(context.Background(),
		awsConfig.WithEndpointResolverWithOptions(r2Resolver), // nolint:staticcheck
		awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			config.AccessKeyID,
			config.SecretAccessKey,
			"",
		)),
		awsConfig.WithRegion(config.Region),
	)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to load aws config"))
	}

	s3Client := s3.NewFromConfig(cfg)

	// Initialize Iceberg REST catalog if configured
	var icebergClient catalog.Catalog
	if config.CatalogEndpoint != "" && config.CatalogToken != "" {
		logger.Info("initializing iceberg REST catalog",
			"endpoint", config.CatalogEndpoint,
		)

		catalogProps := iceberg.Properties{
			"type":      "rest",
			"uri":       config.CatalogEndpoint,
			"token":     config.CatalogToken,
			"warehouse": config.Bucket,
		}

		var catalogErr error
		icebergClient, catalogErr = catalog.Load(context.Background(), "unkey", catalogProps)
		if catalogErr != nil {
			logger.Warn("failed to initialize iceberg catalog, catalog commits will be skipped",
				"error", catalogErr.Error(),
			)
			icebergClient = nil
		} else {
			logger.Info("iceberg catalog initialized successfully")
		}
	}

	writer := &Writer{
		config:          config,
		logger:          logger,
		s3Client:        s3Client,
		icebergClient:   icebergClient,
		circuitBreakers: make(map[string]circuitbreaker.CircuitBreaker[struct{}]),
	}

	// Validate S3 access by testing bucket access (optional, can be disabled for faster startup)
	// This helps catch credential/permission issues early
	if config.AccessKeyID != "" {
		logger.Info("validating s3 access for iceberg writer", "bucket", config.Bucket)
		// Note: Validation will happen on first write attempt
	}

	// Create batch processors for each event type
	writer.keyVerificationProcessor = batch.New(batch.Config[KeyVerificationParquet]{
		Name:          "iceberg_key_verifications",
		Drop:          false, // Block if buffer full to ensure no data loss
		BatchSize:     1000,
		BufferSize:    10000,
		FlushInterval: 30 * time.Second,
		Consumers:     2,
		Flush: func(ctx context.Context, events []KeyVerificationParquet) {
			writer.flushKeyVerifications(ctx, events)
		},
	})

	writer.ratelimitProcessor = batch.New(batch.Config[RatelimitParquet]{
		Name:          "iceberg_ratelimits",
		Drop:          false,
		BatchSize:     1000,
		BufferSize:    10000,
		FlushInterval: 30 * time.Second,
		Consumers:     2,
		Flush: func(ctx context.Context, events []RatelimitParquet) {
			writer.flushRatelimits(ctx, events)
		},
	})

	writer.apiRequestProcessor = batch.New(batch.Config[ApiRequestParquet]{
		Name:          "iceberg_api_requests",
		Drop:          false,
		BatchSize:     1000,
		BufferSize:    10000,
		FlushInterval: 30 * time.Second,
		Consumers:     2,
		Flush: func(ctx context.Context, events []ApiRequestParquet) {
			writer.flushApiRequests(ctx, events)
		},
	})

	return writer, nil
}

// KeyVerification writes a key verification event to the customer's data lake.
func (w *Writer) KeyVerification(ctx context.Context, data schema.KeyVerificationV2) error {
	// Convert to Parquet struct and buffer for batching
	parquetData := KeyVerificationParquet{
		RequestID:    data.RequestID,
		Time:         data.Time,
		WorkspaceID:  data.WorkspaceID,
		KeySpaceID:   data.KeySpaceID,
		IdentityID:   data.IdentityID,
		KeyID:        data.KeyID,
		Region:       data.Region,
		Outcome:      data.Outcome,
		Tags:         data.Tags,
		SpentCredits: data.SpentCredits,
		Latency:      data.Latency,
	}

	w.keyVerificationProcessor.Buffer(parquetData)
	return nil
}

// Ratelimit writes a ratelimit event to the customer's data lake.
func (w *Writer) Ratelimit(ctx context.Context, data schema.RatelimitV2) error {
	// Convert to Parquet struct and buffer for batching
	parquetData := RatelimitParquet{
		RequestID:   data.RequestID,
		Time:        data.Time,
		WorkspaceID: data.WorkspaceID,
		NamespaceID: data.NamespaceID,
		Identifier:  data.Identifier,
		Passed:      data.Passed,
		Latency:     data.Latency,
		OverrideID:  data.OverrideID,
		Limit:       data.Limit,
		Remaining:   data.Remaining,
		ResetAt:     data.ResetAt,
	}

	w.ratelimitProcessor.Buffer(parquetData)
	return nil
}

// ApiRequest writes an API request event to the customer's data lake.
func (w *Writer) ApiRequest(ctx context.Context, data schema.ApiRequestV2) error {
	// Convert QueryParams map to JSON string
	queryParamsJSON, err := json.Marshal(data.QueryParams)
	if err != nil {
		w.logger.Error("failed to marshal query params", "error", err)
		queryParamsJSON = []byte("{}")
	}

	// Convert to Parquet struct and buffer for batching
	parquetData := ApiRequestParquet{
		RequestID:       data.RequestID,
		Time:            data.Time,
		WorkspaceID:     data.WorkspaceID,
		Host:            data.Host,
		Method:          data.Method,
		Path:            data.Path,
		QueryString:     data.QueryString,
		QueryParams:     string(queryParamsJSON),
		RequestHeaders:  data.RequestHeaders,
		RequestBody:     data.RequestBody,
		ResponseStatus:  data.ResponseStatus,
		ResponseHeaders: data.ResponseHeaders,
		ResponseBody:    data.ResponseBody,
		Error:           data.Error,
		ServiceLatency:  data.ServiceLatency,
		UserAgent:       data.UserAgent,
		IpAddress:       data.IpAddress,
		Region:          data.Region,
	}

	w.apiRequestProcessor.Buffer(parquetData)
	return nil
}

// Flush methods called by batch processors

func (w *Writer) flushKeyVerifications(ctx context.Context, events []KeyVerificationParquet) {
	if len(events) == 0 {
		return
	}

	// Group by workspace
	byWorkspace := make(map[string][]KeyVerificationParquet)
	for _, event := range events {
		byWorkspace[event.WorkspaceID] = append(byWorkspace[event.WorkspaceID], event)
	}

	// Write one Parquet file per workspace
	for workspaceID, workspaceEvents := range byWorkspace {
		// Use circuit breaker to fail fast after consecutive errors
		cb := w.getOrCreateCircuitBreaker(workspaceID)
		_, err := cb.Do(ctx, func(ctx context.Context) (struct{}, error) {
			return struct{}{}, w.writeParquetAndCommit(ctx, workspaceID, "key_verifications", workspaceEvents)
		})

		if err != nil {
			if errors.Is(err, circuitbreaker.ErrTripped) {
				w.logger.Warn("circuit breaker open, skipping write to customer data lake",
					"workspace_id", workspaceID,
					"count", len(workspaceEvents),
					"table", "key_verifications",
				)
			} else {
				w.logger.Error("failed to flush key verifications",
					"workspace_id", workspaceID,
					"count", len(workspaceEvents),
					"error", err.Error(),
				)
			}
		}
	}
}

func (w *Writer) flushRatelimits(ctx context.Context, events []RatelimitParquet) {
	if len(events) == 0 {
		return
	}

	// Group by workspace
	byWorkspace := make(map[string][]RatelimitParquet)
	for _, event := range events {
		byWorkspace[event.WorkspaceID] = append(byWorkspace[event.WorkspaceID], event)
	}

	// Write one Parquet file per workspace
	for workspaceID, workspaceEvents := range byWorkspace {
		// Use circuit breaker to fail fast after consecutive errors
		cb := w.getOrCreateCircuitBreaker(workspaceID)
		_, err := cb.Do(ctx, func(ctx context.Context) (struct{}, error) {
			return struct{}{}, w.writeParquetAndCommit(ctx, workspaceID, "ratelimits", workspaceEvents)
		})

		if err != nil {
			if errors.Is(err, circuitbreaker.ErrTripped) {
				w.logger.Warn("circuit breaker open, skipping write to customer data lake",
					"workspace_id", workspaceID,
					"count", len(workspaceEvents),
					"table", "ratelimits",
				)
			} else {
				w.logger.Error("failed to flush ratelimits",
					"workspace_id", workspaceID,
					"count", len(workspaceEvents),
					"error", err.Error(),
				)
			}
		}
	}
}

func (w *Writer) flushApiRequests(ctx context.Context, events []ApiRequestParquet) {
	if len(events) == 0 {
		return
	}

	// Group by workspace
	byWorkspace := make(map[string][]ApiRequestParquet)
	for _, event := range events {
		byWorkspace[event.WorkspaceID] = append(byWorkspace[event.WorkspaceID], event)
	}

	// Write one Parquet file per workspace
	for workspaceID, workspaceEvents := range byWorkspace {
		// Use circuit breaker to fail fast after consecutive errors
		cb := w.getOrCreateCircuitBreaker(workspaceID)
		_, err := cb.Do(ctx, func(ctx context.Context) (struct{}, error) {
			return struct{}{}, w.writeParquetAndCommit(ctx, workspaceID, "api_requests", workspaceEvents)
		})

		if err != nil {
			if errors.Is(err, circuitbreaker.ErrTripped) {
				w.logger.Warn("circuit breaker open, skipping write to customer data lake",
					"workspace_id", workspaceID,
					"count", len(workspaceEvents),
					"table", "api_requests",
				)
			} else {
				w.logger.Error("failed to flush api requests",
					"workspace_id", workspaceID,
					"count", len(workspaceEvents),
					"error", err.Error(),
				)
			}
		}
	}
}

// writeParquetAndCommit writes data to a Parquet file and commits it to Iceberg via REST API
func (w *Writer) writeParquetAndCommit(ctx context.Context, workspaceID, tableName string, data interface{}) error {
	// Step 1: Write data to Parquet format in memory
	var buf bytes.Buffer
	var err error
	var rowCount int

	// Write based on data type
	switch v := data.(type) {
	case []KeyVerificationParquet:
		rowCount = len(v)
		err = parquet.Write(&buf, v)
	case []RatelimitParquet:
		rowCount = len(v)
		err = parquet.Write(&buf, v)
	case []ApiRequestParquet:
		rowCount = len(v)
		err = parquet.Write(&buf, v)
	default:
		return fault.New(fmt.Sprintf("unsupported data type for parquet: %T", data))
	}

	if err != nil {
		return fault.Wrap(err, fault.Internal("failed to write parquet data"))
	}

	w.logger.Debug("wrote parquet data",
		"workspace_id", workspaceID,
		"table", tableName,
		"rows", rowCount,
		"bytes", buf.Len(),
	)

	// Step 2: Upload Parquet file to S3
	key := fmt.Sprintf("data/%s/%d.parquet", tableName, time.Now().UnixNano())

	_, err = w.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(w.config.Bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(buf.Bytes()),
	})
	if err != nil {
		w.logger.Error("failed to upload parquet file to s3",
			"workspace_id", workspaceID,
			"bucket", w.config.Bucket,
			"key", key,
			"endpoint", w.config.Endpoint,
			"has_credentials", w.config.AccessKeyID != "",
			"error", err.Error(),
		)
		return fault.Wrap(err, fault.Internal("failed to upload parquet file to s3"))
	}

	w.logger.Info("uploaded parquet file to s3",
		"workspace_id", workspaceID,
		"bucket", w.config.Bucket,
		"key", key,
		"size", buf.Len(),
	)

	// Step 3: Commit file to Iceberg table via REST Catalog API
	if w.icebergClient != nil {
		s3Location := fmt.Sprintf("s3://%s/%s", w.config.Bucket, key)

		err = w.commitToIcebergCatalog(ctx, workspaceID, tableName, s3Location, rowCount)
		if err != nil {
			w.logger.Error("failed to commit to iceberg catalog (file uploaded but not registered)",
				"workspace_id", workspaceID,
				"table", tableName,
				"file", key,
				"error", err.Error(),
			)
			// Don't return error - file is uploaded, can be registered later
		} else {
			w.logger.Info("committed file to iceberg catalog",
				"workspace_id", workspaceID,
				"table", tableName,
				"file", key,
			)
		}
	}

	return nil
}

// commitToIcebergCatalog commits a Parquet file to an Iceberg table via REST Catalog API
func (w *Writer) commitToIcebergCatalog(ctx context.Context, namespace, tableName, s3Location string, rowCount int) error {
	// Create or load the table
	tableIdent := catalog.ToIdentifier(namespace, tableName)

	// Try to load existing table
	tbl, err := w.icebergClient.LoadTable(ctx, tableIdent, nil)
	if err != nil {
		if errors.Is(err, catalog.ErrNoSuchTable) {
			// Table doesn't exist, create it
			w.logger.Info("creating iceberg table",
				"namespace", namespace,
				"table", tableName,
			)

			// TODO: Define schema based on table type (key_verifications, ratelimits, api_requests)
			// For now, skip table creation - let the catalog handle it or require pre-created tables
			return fmt.Errorf("table does not exist and auto-creation not implemented: %s.%s", namespace, tableName)
		}
		return fmt.Errorf("failed to load table: %w", err)
	}

	w.logger.Debug("loaded iceberg table",
		"namespace", namespace,
		"table", tableName,
		"location", tbl.Location(),
	)

	// Append the data file to the table
	// The REST catalog handles manifest creation, snapshot metadata, etc.
	txn := tbl.NewTransaction()

	// Add the parquet file that was already uploaded to S3
	snapshotProps := iceberg.Properties{
		"row_count": fmt.Sprintf("%d", rowCount),
	}

	err = txn.AddFiles(ctx, []string{s3Location}, snapshotProps, false)
	if err != nil {
		return fmt.Errorf("failed to add file to transaction: %w", err)
	}

	// Commit the transaction - this registers the file in the Iceberg catalog
	_, err = txn.Commit(ctx)
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	w.logger.Info("successfully committed file to iceberg catalog",
		"s3_location", s3Location,
		"row_count", rowCount,
	)

	return nil
}

// getBucketName returns the configured bucket name.
func (w *Writer) getBucketName(workspaceID string) string {
	return w.config.Bucket
}

// getOrCreateCircuitBreaker returns a circuit breaker for the given workspace.
// Circuit breakers are lazily created per workspace to handle credential failures.
func (w *Writer) getOrCreateCircuitBreaker(workspaceID string) circuitbreaker.CircuitBreaker[struct{}] {
	// Check if circuit breaker exists (fast path with read lock)
	w.circuitBreakersMu.RLock()
	cb, ok := w.circuitBreakers[workspaceID]
	w.circuitBreakersMu.RUnlock()
	if ok {
		return cb
	}

	// Create new circuit breaker (slow path with write lock)
	w.circuitBreakersMu.Lock()
	defer w.circuitBreakersMu.Unlock()

	// Double-check after acquiring write lock
	if cb, ok := w.circuitBreakers[workspaceID]; ok {
		return cb
	}

	// Create circuit breaker for this workspace
	// Trip after 5 consecutive failures, stay open for 5 minutes
	cb = circuitbreaker.New[struct{}](
		fmt.Sprintf("iceberg_%s_%s", w.config.Bucket, workspaceID),
		circuitbreaker.WithTripThreshold(5),             // Open after 5 failures
		circuitbreaker.WithTimeout(5*time.Minute),       // Stay open for 5 minutes
		circuitbreaker.WithCyclicPeriod(30*time.Second), // Reset counters every 30s
		circuitbreaker.WithMaxRequests(3),               // Allow 3 requests in half-open state
		circuitbreaker.WithLogger(w.logger),
		circuitbreaker.WithIsDownstreamError(func(err error) bool {
			// All errors count as downstream errors for now
			// The circuit breaker will help us fail fast after repeated S3/credential failures
			return err != nil
		}),
	)

	w.circuitBreakers[workspaceID] = cb
	w.logger.Info("created circuit breaker for workspace", "workspace_id", workspaceID)
	return cb
}

// Close gracefully shuts down the data lake writer.
// Flushes all pending batches before closing.
func (w *Writer) Close(ctx context.Context) error {
	w.logger.Info("closing iceberg writer, flushing pending batches")

	// Close all batch processors (will flush remaining data)
	w.keyVerificationProcessor.Close()
	w.ratelimitProcessor.Close()
	w.apiRequestProcessor.Close()

	w.logger.Info("iceberg writer closed")
	return nil
}
