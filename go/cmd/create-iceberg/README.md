# create-iceberg

Test script for creating and configuring a workspace's Iceberg data lake.

## Purpose

This command:
1. Creates an Iceberg data lake configuration for a workspace
2. Vault-encrypts the access credentials
3. Stores the configuration in the `analytics_config` table
4. Validates the configuration can be loaded

## Usage

### Basic Example (Cloudflare R2)

```bash
./unkey create-iceberg \
  --database-primary="user:pass@localhost:3306/unkey?parseTime=true" \
  --vault-master-keys="your-master-key-1,your-master-key-2" \
  --workspace-id="ws_test123" \
  --provider="r2" \
  --bucket="workspace-ws_test123-analytics" \
  --region="auto" \
  --endpoint="https://abc123.r2.cloudflarestorage.com" \
  --access-key-id="YOUR_R2_ACCESS_KEY_ID" \
  --secret-access-key="YOUR_R2_SECRET_ACCESS_KEY"
```

### AWS S3 Example

```bash
./unkey create-iceberg \
  --database-primary="user:pass@localhost:3306/unkey?parseTime=true" \
  --vault-master-keys="your-master-key-1,your-master-key-2" \
  --workspace-id="ws_test456" \
  --provider="s3" \
  --bucket="my-workspace-analytics" \
  --region="us-east-1" \
  --access-key-id="YOUR_AWS_ACCESS_KEY_ID" \
  --secret-access-key="YOUR_AWS_SECRET_ACCESS_KEY"
```

### Using Environment Variables

```bash
export UNKEY_DATABASE_PRIMARY="user:pass@localhost:3306/unkey?parseTime=true"
export UNKEY_VAULT_MASTER_KEYS="key1,key2"
export WORKSPACE_ID="ws_test789"
export STORAGE_PROVIDER="r2"
export BUCKET="my-analytics-bucket"
export ENDPOINT="https://abc123.r2.cloudflarestorage.com"
export ACCESS_KEY_ID="your-access-key-id"
export SECRET_ACCESS_KEY="your-secret-access-key"

./unkey create-iceberg
```

## What It Does

1. **Connects to Database**: Establishes connection to MySQL to store the configuration
2. **Initializes Vault**: Sets up vault service for encrypting credentials
3. **Encrypts Credentials**: Vault-encrypts the `access-key-id` and `secret-access-key`
4. **Creates Configuration**: Builds an `IcebergWorkspaceConfig` with encrypted credentials
5. **Stores in Database**: Inserts/updates the configuration in `analytics_config` table
6. **Prints Summary**: Displays the configuration for verification

## Configuration Storage

The configuration is stored in the `analytics_config` table as:

```json
{
  "provider": "r2",
  "bucket": "workspace-ws_test123-analytics",
  "region": "auto",
  "endpoint": "https://abc123.r2.cloudflarestorage.com",
  "encryptedAccessKeyId": "vault:encrypted:...",
  "encryptedSecretAccessKey": "vault:encrypted:...",
  "format": "iceberg"
}
```

Note: The `encryptedAccessKeyId` and `encryptedSecretAccessKey` fields contain vault-encrypted values.

## Testing

After running this command:

1. Start `konsume` with the same vault master keys
2. Send analytics events for the configured workspace
3. Events will be written to:
   - Unkey's ClickHouse (primary)
   - The workspace's custom Iceberg data lake (secondary)

## Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--database-primary` | Yes | - | MySQL connection string |
| `--vault-master-keys` | Yes | - | Vault master keys (comma-separated) |
| `--workspace-id` | No | `test_workspace` | Workspace ID to configure |
| `--provider` | No | `r2` | Storage provider (r2, s3, gcs, azure) |
| `--bucket` | Yes | - | Bucket name for analytics data |
| `--region` | No | `auto` | Region code |
| `--endpoint` | No | - | Custom endpoint URL (required for R2) |
| `--access-key-id` | Yes | - | Access key ID for authentication |
| `--secret-access-key` | Yes | - | Secret access key for authentication |
| `--format` | No | `iceberg` | Data format (iceberg, parquet) |

## Updating Configuration

Running the command again with the same `workspace-id` will update the existing configuration (upsert behavior).

## Security Notes

- Credentials are vault-encrypted before storage
- Never commit actual credentials to version control
- Use environment variables for sensitive values in production
- Rotate credentials regularly