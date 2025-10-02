package konsume

import (
	"github.com/unkeyed/unkey/go/pkg/assert"
)

type VaultS3Config struct {
	S3URL             string
	S3Bucket          string
	S3AccessKeyID     string
	S3AccessKeySecret string
}

type Config struct {
	// InstanceID is the unique identifier for this instance of the consumer
	InstanceID string

	// Platform identifies the cloud platform where the node is running (e.g., aws, gcp, hetzner)
	Platform string

	// Image specifies the container image identifier including repository and tag
	Image string

	// Region identifies the geographic region where this node is deployed
	Region string

	// --- Kafka configuration ---

	// KafkaBrokers is the list of Kafka broker addresses
	KafkaBrokers []string

	// ConsumerGroup is the Kafka consumer group ID for this instance
	ConsumerGroup string

	// Topics defines which Kafka topics to consume from
	Topics Topics

	// --- Database configuration ---

	// DatabasePrimary is the primary database connection string for reading workspace configs
	DatabasePrimary string

	// DatabaseReadonlyReplica is an optional read-replica database connection string
	DatabaseReadonlyReplica string

	// --- Vault configuration ---

	// VaultMasterKeys for decrypting sensitive config fields
	VaultMasterKeys []string

	// VaultS3 configuration for vault storage
	VaultS3 *VaultS3Config

	// --- Analytics Writer Configuration ---

	// ClickhouseURL is the ClickHouse connection string for Unkey's analytics
	ClickhouseURL string

	// --- OpenTelemetry configuration ---

	// Enable sending otel data to the collector endpoint for metrics, traces, and logs
	OtelEnabled           bool
	OtelTraceSamplingRate float64

	// PrometheusPort for metrics endpoint
	PrometheusPort int
}

// Topics contains the Kafka topic names for different event types
type Topics struct {
	// KeyVerifications is the topic for key verification events
	KeyVerifications string

	// Ratelimits is the topic for ratelimit events
	Ratelimits string

	// ApiRequests is the topic for API request events
	ApiRequests string
}

func (c Config) Validate() error {
	err := assert.All(
		assert.NotEmpty(c.InstanceID, "instance-id is required"),
		assert.NotEmpty(c.ConsumerGroup, "consumer-group is required"),
		assert.True(len(c.KafkaBrokers) > 0, "kafka-brokers must not be empty"),
		assert.NotEmpty(c.Topics.KeyVerifications, "key-verifications-topic is required"),
		assert.NotEmpty(c.Topics.Ratelimits, "ratelimits-topic is required"),
		assert.NotEmpty(c.Topics.ApiRequests, "api-requests-topic is required"),
		assert.NotEmpty(c.DatabasePrimary, "database-primary is required for loading workspace analytics configs"),
		assert.True(len(c.VaultMasterKeys) > 0, "vault-master-keys is required for decrypting workspace configs"),
		assert.NotNil(c.VaultS3, "vault s3 configuration is required for decrypting workspace configs"),
		assert.NotEmpty(c.ClickhouseURL, "clickhouse-url is required"),
	)
	if err != nil {
		return err
	}

	if c.VaultS3 != nil {
		return assert.All(
			assert.NotEmpty(c.VaultS3.S3URL, "vault s3 url is required"),
			assert.NotEmpty(c.VaultS3.S3Bucket, "vault s3 bucket is required"),
			assert.NotEmpty(c.VaultS3.S3AccessKeyID, "vault s3 access key id is required"),
			assert.NotEmpty(c.VaultS3.S3AccessKeySecret, "vault s3 secret access key is required"),
		)
	}

	return nil
}
