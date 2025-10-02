package konsume

import (
	"context"

	"github.com/unkeyed/unkey/go/apps/konsume"
	"github.com/unkeyed/unkey/go/pkg/cli"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

var Cmd = &cli.Command{
	Name:  "konsume",
	Usage: "Run the Unkey analytics ingestion pipeline (Kafka consumer -> ClickHouse/Data Lakes)",

	Flags: []cli.Flag{
		// Instance Configuration
		cli.String("platform", "Cloud platform identifier for this node. Used for logging and metrics.",
			cli.EnvVar("UNKEY_PLATFORM")),
		cli.String("image", "Container image identifier. Used for logging and metrics.",
			cli.EnvVar("UNKEY_IMAGE")),
		cli.String("region", "Geographic region identifier. Used for logging and routing. Default: unknown",
			cli.Default("unknown"), cli.EnvVar("UNKEY_REGION"), cli.EnvVar("AWS_REGION")),
		cli.String("instance-id", "Unique identifier for this instance. Auto-generated if not provided.",
			cli.Default(uid.New(uid.InstancePrefix, 4)), cli.EnvVar("UNKEY_INSTANCE_ID")),

		// Kafka Consumer Configuration
		cli.StringSlice("kafka-brokers", "Comma-separated list of Kafka broker addresses. Required.",
			cli.Required(), cli.EnvVar("UNKEY_KAFKA_BROKERS")),
		cli.String("consumer-group", "Kafka consumer group ID. Default: konsume",
			cli.Default("konsume"), cli.EnvVar("UNKEY_CONSUMER_GROUP")),

		// Kafka Topics
		cli.String("key-verifications-topic", "Kafka topic for key verification events. Default: analytics.key_verifications",
			cli.Default("analytics.key_verifications"), cli.EnvVar("UNKEY_KEY_VERIFICATIONS_TOPIC")),
		cli.String("ratelimits-topic", "Kafka topic for ratelimit events. Default: analytics.ratelimits",
			cli.Default("analytics.ratelimits"), cli.EnvVar("UNKEY_RATELIMITS_TOPIC")),
		cli.String("api-requests-topic", "Kafka topic for API request events. Default: analytics.api_requests",
			cli.Default("analytics.api_requests"), cli.EnvVar("UNKEY_API_REQUESTS_TOPIC")),

		// Database Configuration
		cli.String("database-primary", "MySQL connection string for primary database. Required for loading workspace analytics configs. Example: user:pass@host:3306/unkey?parseTime=true",
			cli.Required(), cli.EnvVar("UNKEY_DATABASE_PRIMARY")),
		cli.String("database-replica", "MySQL connection string for read-replica. Format same as database-primary.",
			cli.EnvVar("UNKEY_DATABASE_REPLICA")),

		// Vault Configuration
		cli.StringSlice("vault-master-keys", "Vault master keys for decrypting workspace configs. Required.",
			cli.Required(), cli.EnvVar("UNKEY_VAULT_MASTER_KEYS")),
		cli.String("vault-s3-url", "Vault S3 storage URL. Required.",
			cli.Required(), cli.EnvVar("UNKEY_VAULT_S3_URL")),
		cli.String("vault-s3-bucket", "Vault S3 storage bucket. Required.",
			cli.Required(), cli.EnvVar("UNKEY_VAULT_S3_BUCKET")),
		cli.String("vault-s3-access-key-id", "Vault S3 storage access key ID. Required.",
			cli.Required(), cli.EnvVar("UNKEY_VAULT_S3_ACCESS_KEY_ID")),
		cli.String("vault-s3-access-key-secret", "Vault S3 storage access key secret. Required.",
			cli.Required(), cli.EnvVar("UNKEY_VAULT_S3_ACCESS_KEY_SECRET")),

		// ClickHouse Configuration - Unkey's primary analytics storage
		cli.String("clickhouse-url", "ClickHouse connection string for Unkey analytics. Required. Example: clickhouse://user:pass@host:9000/unkey",
			cli.Required(), cli.EnvVar("UNKEY_CLICKHOUSE_URL")),

		// Observability
		cli.Bool("otel", "Enable OpenTelemetry tracing and metrics",
			cli.EnvVar("UNKEY_OTEL")),
		cli.Float("otel-trace-sampling-rate", "Sampling rate for OpenTelemetry traces (0.0-1.0). Only used when --otel is provided. Default: 0.25",
			cli.Default(0.25), cli.EnvVar("UNKEY_OTEL_TRACE_SAMPLING_RATE")),
		cli.Int("prometheus-port", "Enable Prometheus /metrics endpoint on specified port. Set to 0 to disable.",
			cli.Default(0), cli.EnvVar("UNKEY_PROMETHEUS_PORT")),
	},

	Action: action,
}

func action(ctx context.Context, cmd *cli.Command) error {
	config := konsume.Config{
		Platform:   cmd.String("platform"),
		Image:      cmd.String("image"),
		Region:     cmd.String("region"),
		InstanceID: cmd.String("instance-id"),

		KafkaBrokers:  cmd.StringSlice("kafka-brokers"),
		ConsumerGroup: cmd.String("consumer-group"),

		Topics: konsume.Topics{
			KeyVerifications: cmd.String("key-verifications-topic"),
			Ratelimits:       cmd.String("ratelimits-topic"),
			ApiRequests:      cmd.String("api-requests-topic"),
		},

		DatabasePrimary:         cmd.String("database-primary"),
		DatabaseReadonlyReplica: cmd.String("database-replica"),

		VaultMasterKeys: cmd.StringSlice("vault-master-keys"),
		VaultS3: &konsume.VaultS3Config{
			S3URL:             cmd.String("vault-s3-url"),
			S3Bucket:          cmd.String("vault-s3-bucket"),
			S3AccessKeyID:     cmd.String("vault-s3-access-key-id"),
			S3AccessKeySecret: cmd.String("vault-s3-access-key-secret"),
		},

		ClickhouseURL: cmd.String("clickhouse-url"),

		OtelEnabled:           cmd.Bool("otel"),
		OtelTraceSamplingRate: cmd.Float("otel-trace-sampling-rate"),
		PrometheusPort:        cmd.Int("prometheus-port"),
	}

	err := config.Validate()
	if err != nil {
		return err
	}

	return konsume.Run(ctx, config)
}
