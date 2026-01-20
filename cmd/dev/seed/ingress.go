package seed

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"time"

	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/vault"
	"github.com/unkeyed/unkey/pkg/vault/storage"
)

var frontlineCmd = &cli.Command{
	Name:  "frontline",
	Usage: "Seed database with deployment, sentinel, instance, frontline route, and TLS certificate for testing frontline/sentinel",
	Flags: []cli.Flag{
		cli.String("database-primary", "MySQL database DSN", cli.Default("unkey:password@tcp(127.0.0.1:3306)/unkey?parseTime=true&interpolateParams=true"), cli.EnvVar("UNKEY_DATABASE_PRIMARY")),
		cli.String("slug", "Slug to match local seed (e.g., 'local' uses ws_local, proj_local, etc.)", cli.Default("local")),
		cli.String("hostname", "Hostname for frontline route", cli.Default("unkey.local")),
		cli.String("region", "Region for sentinel and instance", cli.Default("local")),
		cli.String("address", "Address for instance (IP or hostname)", cli.Default("127.0.0.1:8787")),
		cli.String("vault-s3-url", "Vault S3 URL", cli.Default("http://127.0.0.1:3902"), cli.EnvVar("UNKEY_VAULT_S3_URL")),
		cli.String("vault-s3-bucket", "Vault S3 bucket", cli.Default("vault"), cli.EnvVar("UNKEY_VAULT_S3_BUCKET")),
		cli.String("vault-s3-access-key-id", "Vault S3 access key ID", cli.Default("minio_root_user"), cli.EnvVar("UNKEY_VAULT_S3_ACCESS_KEY_ID")),
		cli.String("vault-s3-access-key-secret", "Vault S3 access key secret", cli.Default("minio_root_password"), cli.EnvVar("UNKEY_VAULT_S3_ACCESS_KEY_SECRET")),
		cli.String("vault-master-keys", "Vault master keys (comma-separated)", cli.Default("Ch9rZWtfMmdqMFBJdVhac1NSa0ZhNE5mOWlLSnBHenFPENTt7an5MRogENt9Si6wms4pQ2XIvqNSIgNpaBenJmXgcInhu6Nfv2U="), cli.EnvVar("UNKEY_VAULT_MASTER_KEYS")),
	},
	Action: seedFrontline,
}

func seedFrontline(ctx context.Context, cmd *cli.Command) error {
	logger := logging.New()

	database, err := db.New(db.Config{
		PrimaryDSN:  cmd.RequireString("database-primary"),
		ReadOnlyDSN: "",
		Logger:      logger,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	s3Storage, err := storage.NewS3(storage.S3Config{
		S3URL:             cmd.String("vault-s3-url"),
		S3Bucket:          cmd.String("vault-s3-bucket"),
		S3AccessKeyID:     cmd.String("vault-s3-access-key-id"),
		S3AccessKeySecret: cmd.String("vault-s3-access-key-secret"),
		Logger:            logger,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize S3 storage: %w", err)
	}

	vaultService, err := vault.New(vault.Config{
		Logger:     logger,
		Storage:    s3Storage,
		MasterKeys: []string{cmd.String("vault-master-keys")},
	})
	if err != nil {
		return fmt.Errorf("failed to initialize vault: %w", err)
	}

	slug := cmd.String("slug")
	hostname := cmd.String("hostname")
	region := cmd.String("region")
	address := cmd.String("address")
	now := time.Now().UnixMilli()

	workspaceID := fmt.Sprintf("ws_%s", slug)
	projectID := fmt.Sprintf("proj_%s", slug)
	envID := fmt.Sprintf("env_%s", slug)

	deploymentID := uid.New(uid.DeploymentPrefix)
	sentinelID := uid.New(uid.SentinelPrefix)
	instanceID := uid.New(uid.InstancePrefix)
	frontlineRouteID := uid.New(uid.FrontlineRoutePrefix)
	certificateID := uid.New(uid.CertificatePrefix)

	certPEM, keyPEM, err := generateMkcertCertificate(hostname)
	if err != nil {
		return fmt.Errorf("failed to generate certificate: %w", err)
	}

	encryptResp, err := vaultService.Encrypt(ctx, &vaultv1.EncryptRequest{
		Keyring: "unkey_internal",
		Data:    string(keyPEM),
	})
	if err != nil {
		return fmt.Errorf("failed to encrypt private key: %w", err)
	}

	err = db.TxRetry(ctx, database.RW(), func(ctx context.Context, tx db.DBTX) error {
		err := db.Query.InsertDeployment(ctx, tx, db.InsertDeploymentParams{
			ID:                            deploymentID,
			K8sName:                       uid.DNS1035(12),
			WorkspaceID:                   workspaceID,
			ProjectID:                     projectID,
			EnvironmentID:                 envID,
			GitCommitSha:                  sql.NullString{String: "abc123", Valid: true},
			GitBranch:                     sql.NullString{String: "main", Valid: true},
			SentinelConfig:                []byte("{}"),
			GitCommitMessage:              sql.NullString{String: "Local dev seed", Valid: true},
			GitCommitAuthorHandle:         sql.NullString{String: "local", Valid: true},
			GitCommitAuthorAvatarUrl:      sql.NullString{},
			GitCommitTimestamp:            sql.NullInt64{Int64: now, Valid: true},
			OpenapiSpec:                   sql.NullString{},
			EncryptedEnvironmentVariables: []byte("{}"),
			Command:                       []byte("[]"),
			Status:                        db.DeploymentsStatusReady,
			CpuMillicores:                 256,
			MemoryMib:                     256,
			CreatedAt:                     now,
			UpdatedAt:                     sql.NullInt64{Valid: false, Int64: 0},
		})
		if err != nil && !db.IsDuplicateKeyError(err) {
			return fmt.Errorf("failed to create deployment: %w", err)
		}

		sentinelName := uid.DNS1035()
		err = db.Query.InsertSentinel(ctx, tx, db.InsertSentinelParams{
			ID:                sentinelID,
			WorkspaceID:       workspaceID,
			EnvironmentID:     envID,
			K8sAddress:        fmt.Sprintf("%s.%s.svc.cluster.local", sentinelName, "todonamespace"),
			K8sName:           sentinelName,
			Region:            region,
			Image:             "unkey/sentinel:local",
			Health:            db.SentinelsHealthHealthy,
			DesiredReplicas:   1,
			AvailableReplicas: 0,
			ProjectID:         projectID,
			CpuMillicores:     512,
			MemoryMib:         512,
			Version:           uint64(now),
			CreatedAt:         now,
		})
		if err != nil && !db.IsDuplicateKeyError(err) {
			return fmt.Errorf("failed to create sentinel: %w", err)
		}

		err = db.Query.InsertFrontlineRoute(ctx, tx, db.InsertFrontlineRouteParams{
			ID:                       frontlineRouteID,
			ProjectID:                projectID,
			DeploymentID:             deploymentID,
			EnvironmentID:            envID,
			FullyQualifiedDomainName: hostname,
			Sticky:                   db.FrontlineRoutesStickyLive,
			CreatedAt:                now,
			UpdatedAt:                sql.NullInt64{},
		})
		if err != nil && !db.IsDuplicateKeyError(err) {
			return fmt.Errorf("failed to create frontline route: %w", err)
		}

		err = db.Query.InsertCertificate(ctx, tx, db.InsertCertificateParams{
			ID:                  certificateID,
			WorkspaceID:         workspaceID,
			Hostname:            hostname,
			Certificate:         string(certPEM),
			EncryptedPrivateKey: encryptResp.GetEncrypted(),
			CreatedAt:           now,
			UpdatedAt:           sql.NullInt64{Valid: true, Int64: now},
		})
		if err != nil && !db.IsDuplicateKeyError(err) {
			return fmt.Errorf("failed to create certificate: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	logger.Info("seed completed",
		"deployment", deploymentID,
		"sentinel", sentinelID,
		"instance", instanceID,
		"frontlineRoute", frontlineRouteID,
		"certificate", certificateID,
		"hostname", hostname,
		"address", address,
	)

	return nil
}

func generateMkcertCertificate(hostname string) (certPEM []byte, keyPEM []byte, err error) {
	if _, err = exec.LookPath("mkcert"); err != nil {
		return nil, nil, fmt.Errorf("mkcert not found - install with: brew install mkcert (or visit https://github.com/FiloSottile/mkcert)")
	}

	tempDir, err := os.MkdirTemp("", "unkey-certs-*")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	certFile := fmt.Sprintf("%s/%s.pem", tempDir, hostname)
	keyFile := fmt.Sprintf("%s/%s-key.pem", tempDir, hostname)

	cmd := exec.Command("mkcert", "-cert-file", certFile, "-key-file", keyFile, hostname, "*."+hostname)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, nil, fmt.Errorf("mkcert failed: %w\nOutput: %s", err, string(output))
	}

	certPEM, err = os.ReadFile(certFile)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read certificate: %w", err)
	}

	keyPEM, err = os.ReadFile(keyFile)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read private key: %w", err)
	}

	return certPEM, keyPEM, nil
}
