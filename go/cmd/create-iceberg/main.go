package createiceberg

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/cloudflare/cloudflare-go"
	vaultv1 "github.com/unkeyed/unkey/go/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/go/pkg/analytics/storage/iceberg"
	"github.com/unkeyed/unkey/go/pkg/cli"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/vault"
	"github.com/unkeyed/unkey/go/pkg/vault/storage"
)

var Cmd = &cli.Command{
	Name:  "create-iceberg",
	Usage: "Create and configure an Iceberg data lake for a workspace (for testing)",

	Flags: []cli.Flag{
		// Database configuration
		cli.String("database-primary", "MySQL connection string for primary database",
			cli.Required(), cli.EnvVar("UNKEY_DATABASE_PRIMARY")),

		// Vault configuration
		cli.StringSlice("vault-master-keys", "Vault master keys for encrypting sensitive config fields",
			cli.Required(), cli.EnvVar("UNKEY_VAULT_MASTER_KEYS")),
		cli.String("vault-s3-url", "Vault S3 storage URL",
			cli.Required(), cli.EnvVar("UNKEY_VAULT_S3_URL")),
		cli.String("vault-s3-bucket", "Vault S3 storage bucket",
			cli.Required(), cli.EnvVar("UNKEY_VAULT_S3_BUCKET")),
		cli.String("vault-s3-access-key-id", "Vault S3 storage access key ID",
			cli.Required(), cli.EnvVar("UNKEY_VAULT_S3_ACCESS_KEY_ID")),
		cli.String("vault-s3-access-key-secret", "Vault S3 storage access key secret",
			cli.Required(), cli.EnvVar("UNKEY_VAULT_S3_ACCESS_KEY_SECRET")),

		// Workspace configuration
		cli.String("workspace-id", "Workspace ID to configure Iceberg for",
			cli.Required(), cli.EnvVar("WORKSPACE_ID")),

		// Cloudflare R2 configuration
		cli.String("cloudflare-account-id", "Cloudflare account ID",
			cli.Required(), cli.EnvVar("CLOUDFLARE_ACCOUNT_ID")),
		cli.String("cloudflare-api-token", "Cloudflare API token",
			cli.Required(), cli.EnvVar("CLOUDFLARE_API_TOKEN")),

		// Data lake configuration
		cli.String("bucket-prefix", "Bucket prefix. Bucket name will be {prefix}-{workspace_id}",
			cli.Default("unkey-analytics"), cli.EnvVar("BUCKET_PREFIX")),
		cli.String("format", "Data format (iceberg, parquet)",
			cli.Default("iceberg"), cli.EnvVar("FORMAT")),
		cli.Int("retention-days", "Number of days to retain data (snapshots)",
			cli.Default(30), cli.EnvVar("RETENTION_DAYS")),
	},

	Action: action,
}

func sanitizeBucketName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "_", "-")
	return name
}

func action(ctx context.Context, cmd *cli.Command) error {
	logger := logging.New()

	database, err := db.New(db.Config{
		PrimaryDSN: cmd.String("database-primary"),
		Logger:     logger,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer database.Close()

	vaultStorage, err := storage.NewS3(storage.S3Config{
		Logger:            logger,
		S3URL:             cmd.String("vault-s3-url"),
		S3Bucket:          cmd.String("vault-s3-bucket"),
		S3AccessKeyID:     cmd.String("vault-s3-access-key-id"),
		S3AccessKeySecret: cmd.String("vault-s3-access-key-secret"),
	})
	if err != nil {
		return fmt.Errorf("failed to create vault storage: %w", err)
	}

	vaultService, err := vault.New(vault.Config{
		Logger:     logger,
		Storage:    vaultStorage,
		MasterKeys: cmd.StringSlice("vault-master-keys"),
	})
	if err != nil {
		return fmt.Errorf("failed to initialize vault: %w", err)
	}

	workspaceID := cmd.String("workspace-id")
	bucketName := sanitizeBucketName(fmt.Sprintf("%s-%s", cmd.String("bucket-prefix"), workspaceID))
	cfAccountID := cmd.String("cloudflare-account-id")

	logger.Info("creating cloudflare client")
	cf, err := cloudflare.NewWithAPIToken(cmd.String("cloudflare-api-token"))
	if err != nil {
		return fmt.Errorf("failed to create cloudflare client: %w", err)
	}

	logger.Info("creating R2 bucket", "bucket", bucketName)
	accountID := cloudflare.AccountIdentifier(cfAccountID)
	_, err = cf.CreateR2Bucket(ctx, accountID, cloudflare.CreateR2BucketParameters{
		Name: bucketName,
	})
	if err != nil && !strings.Contains(err.Error(), "(10004)") {
		return fmt.Errorf("failed to create R2 bucket: %w", err)
	}
	logger.Info("R2 bucket created or already exists")

	logger.Info("enabling R2 data catalog")
	catalogEndpoint := fmt.Sprintf("/accounts/%s/r2-catalog/%s/enable", cfAccountID, bucketName)
	catalogResp, err := cf.Raw(ctx, "POST", catalogEndpoint, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to enable R2 data catalog: %w", err)
	}
	logger.Info("R2 data catalog enabled")

	if !catalogResp.Success {
		return fmt.Errorf("failed to enable catalog: %v", catalogResp.Errors)
	}

	var catalogResult struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(catalogResp.Result, &catalogResult); err != nil {
		return fmt.Errorf("failed to parse catalog enable response: %w", err)
	}

	catalogID := catalogResult.ID

	// Fetch IAM permission groups from the account
	logger.Info("fetching account IAM permission groups")
	permGroupsEndpoint := fmt.Sprintf("/accounts/%s/tokens/permission_groups", cfAccountID)
	permGroupsResp, err := cf.Raw(ctx, "GET", permGroupsEndpoint, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to fetch permission groups: %w", err)
	}

	if !permGroupsResp.Success {
		return fmt.Errorf("failed to fetch permission groups: %v", permGroupsResp.Errors)
	}

	var permissionGroups []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(permGroupsResp.Result, &permissionGroups); err != nil {
		return fmt.Errorf("failed to parse permission groups: %w", err)
	}

	// Find R2 permission groups
	var r2ReadID, r2AdminID string
	for _, pg := range permissionGroups {
		switch pg.Name {
		case "Workers R2 Storage Read", "Cloudflare R2 Read":
			r2ReadID = pg.ID
			logger.Info("found R2 read permission", "id", pg.ID, "name", pg.Name)
		case "Workers R2 Storage Write", "Cloudflare R2 Admin":
			r2AdminID = pg.ID
			logger.Info("found R2 admin permission", "id", pg.ID, "name", pg.Name)
		}
	}

	if r2ReadID == "" || r2AdminID == "" {
		logger.Error("could not find required R2 permission groups",
			"available_groups", permissionGroups,
		)
		return fmt.Errorf("could not find required R2 permission groups (read: %s, admin: %s)", r2ReadID, r2AdminID)
	}

	// Create R2 API token scoped to this bucket
	tokenName := fmt.Sprintf("unkey-analytics-%s", workspaceID)
	r2BucketResource := fmt.Sprintf("com.cloudflare.edge.r2.bucket.%s_default_%s", cfAccountID, bucketName)

	tokenPayload := map[string]interface{}{
		"name": tokenName,
		"policies": []map[string]interface{}{
			{
				"effect": "allow",
				"resources": map[string]interface{}{
					r2BucketResource: "*",
				},
				"permission_groups": []map[string]interface{}{
					{
						"id":   r2ReadID,
						"meta": map[string]interface{}{},
					},
					{
						"id":   r2AdminID,
						"meta": map[string]interface{}{},
					},
				},
			},
		},
	}

	logger.Info("creating account R2 API token", "tokenName", tokenName)

	// Create account-level token
	tokenEndpoint := fmt.Sprintf("/accounts/%s/tokens", cfAccountID)
	tokenResp, err := cf.Raw(ctx, "POST", tokenEndpoint, tokenPayload, nil)
	if err != nil {
		return fmt.Errorf("failed to create R2 API token: %w", err)
	}

	if !tokenResp.Success {
		errorsJSON, _ := json.MarshalIndent(tokenResp.Errors, "", "  ")
		logger.Error("token creation failed",
			"errors", string(errorsJSON),
			"messages", tokenResp.Messages,
		)
		return fmt.Errorf("failed to create token: %v", tokenResp.Errors)
	}
	logger.Info("R2 API token created successfully")

	var tokenResult struct {
		ID    string `json:"id"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal(tokenResp.Result, &tokenResult); err != nil {
		return fmt.Errorf("failed to parse token response: %w", err)
	}

	// Convert token to S3-compatible credentials
	accessKeyID := tokenResult.ID
	secretHash := sha256.Sum256([]byte(tokenResult.Value))
	secretAccessKey := hex.EncodeToString(secretHash[:])

	logger.Info("created S3 credentials from R2 token")

	// Wait for token to propagate in Cloudflare's systems
	logger.Info("waiting 10 seconds for token to propagate...")
	time.Sleep(10 * time.Second)

	// Test the credentials before storing them
	accountEndpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfAccountID)
	logger.Info("testing S3 credentials", "bucket", bucketName)

	// nolint:staticcheck
	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...any) (aws.Endpoint, error) {
		// nolint:staticcheck
		return aws.Endpoint{
			URL:               accountEndpoint,
			HostnameImmutable: true,
		}, nil
	})

	cfg, err := awsConfig.LoadDefaultConfig(ctx,
		awsConfig.WithEndpointResolverWithOptions(r2Resolver), // nolint:staticcheck
		awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")),
		awsConfig.WithRegion("auto"),
	)
	if err != nil {
		return fmt.Errorf("failed to load aws config: %w", err)
	}

	s3Client := s3.NewFromConfig(cfg)

	// Try to list objects in the bucket to verify credentials work
	_, err = s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return fmt.Errorf("S3 credentials verification failed (list): %w", err)
	}

	logger.Info("S3 list test passed")

	// Try to write a test object to verify write permissions
	testKey := "test/.credentials-test"
	testData := []byte("test")
	_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(testKey),
		Body:   bytes.NewReader(testData),
	})
	if err != nil {
		return fmt.Errorf("S3 credentials verification failed (write): %w", err)
	}

	logger.Info("S3 write test passed")

	// Clean up test object
	_, err = s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(testKey),
	})
	if err != nil {
		logger.Warn("failed to delete test object", "key", testKey, "error", err.Error())
	}

	logger.Info("S3 credentials verified successfully (read and write)")

	logger.Info("encrypting credentials with vault")
	encryptedAccessKeyIDResp, err := vaultService.Encrypt(ctx, &vaultv1.EncryptRequest{
		Keyring: workspaceID,
		Data:    accessKeyID,
	})
	if err != nil {
		return fmt.Errorf("failed to encrypt access key ID: %w", err)
	}

	encryptedSecretAccessKeyResp, err := vaultService.Encrypt(ctx, &vaultv1.EncryptRequest{
		Keyring: workspaceID,
		Data:    secretAccessKey,
	})
	if err != nil {
		return fmt.Errorf("failed to encrypt secret access key: %w", err)
	}

	// Encrypt catalog token (R2 API token value for catalog authentication)
	encryptedCatalogTokenResp, err := vaultService.Encrypt(ctx, &vaultv1.EncryptRequest{
		Keyring: workspaceID,
		Data:    tokenResult.Value,
	})
	if err != nil {
		return fmt.Errorf("failed to encrypt catalog token: %w", err)
	}

	logger.Info("credentials encrypted")

	// R2 catalog endpoint format: https://catalog.cloudflarestorage.com/{account_id}/{bucket_name}
	r2CatalogURL := fmt.Sprintf("https://catalog.cloudflarestorage.com/%s/%s", cfAccountID, bucketName)

	config := iceberg.WorkspaceConfig{
		Bucket:                   bucketName,
		Region:                   "auto",
		Endpoint:                 accountEndpoint,
		CatalogEndpoint:          r2CatalogURL,
		EncryptedAccessKeyId:     encryptedAccessKeyIDResp.Encrypted,
		EncryptedSecretAccessKey: encryptedSecretAccessKeyResp.Encrypted,
		EncryptedCatalogToken:    encryptedCatalogTokenResp.Encrypted,
		Format:                   cmd.String("format"),
		RetentionDays:            cmd.Int("retention-days"),
		CatalogID:                catalogID,
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	err = db.Query.UpsertAnalyticsConfig(ctx, database.RW(), db.UpsertAnalyticsConfigParams{
		WorkspaceID: workspaceID,
		Storage:     db.AnalyticsConfigStorageIceberg,
		Config:      configJSON,
	})
	if err != nil {
		return fmt.Errorf("failed to upsert config: %w", err)
	}

	logger.Info("created iceberg configuration",
		"workspace", workspaceID,
		"bucket", bucketName,
		"endpoint", accountEndpoint,
		"catalogEndpoint", r2CatalogURL,
		"catalogId", catalogID,
	)

	return nil
}
