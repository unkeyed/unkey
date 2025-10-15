package acme

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
	vaultv1 "github.com/unkeyed/unkey/go/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/vault"
)

type AcmeUser struct {
	WorkspaceID  string
	Registration *registration.Resource
	key          crypto.PrivateKey
}

func (u *AcmeUser) GetEmail() string {
	return fmt.Sprintf("%s@%s", u.WorkspaceID, "unkey.fun")
}

func (u AcmeUser) GetRegistration() *registration.Resource {
	return u.Registration
}

func (u *AcmeUser) GetPrivateKey() crypto.PrivateKey {
	return u.key
}

type UserConfig struct {
	DB          db.Database
	Logger      logging.Logger
	Vault       *vault.Service
	WorkspaceID string
}

func GetOrCreateUser(ctx context.Context, cfg UserConfig) (*lego.Client, error) {
	foundUser, err := db.Query.FindAcmeUserByWorkspaceID(ctx, cfg.DB.RO(), cfg.WorkspaceID)
	if err != nil {
		if db.IsNotFound(err) {
			return register(ctx, cfg)
		}
	}

	resp, err := cfg.Vault.Decrypt(ctx, &vaultv1.DecryptRequest{
		Keyring:   cfg.WorkspaceID,
		Encrypted: foundUser.EncryptedKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt private key: %w", err)
	}

	key, err := stringToPrivateKey(resp.GetPlaintext())
	if err != nil {
		return nil, fmt.Errorf("failed to convert private key: %w", err)
	}

	config := lego.NewConfig(&AcmeUser{
		Registration: &registration.Resource{
			URI: foundUser.RegistrationUri.String,
		},
		key:         key,
		WorkspaceID: cfg.WorkspaceID,
	})
	client, err := lego.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create ACME client: %w", err)
	}

	return client, nil
}

func register(ctx context.Context, cfg UserConfig) (*lego.Client, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	user := AcmeUser{
		Registration: nil,
		key:          privateKey,
		WorkspaceID:  cfg.WorkspaceID,
	}

	privKeyString, err := privateKeyToString(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize private key: %w", err)
	}

	resp, err := cfg.Vault.Encrypt(ctx, &vaultv1.EncryptRequest{
		Keyring: cfg.WorkspaceID,
		Data:    privKeyString,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt private key: %w", err)
	}

	id, err := db.Query.InsertAcmeUser(ctx, cfg.DB.RW(), db.InsertAcmeUserParams{
		WorkspaceID:  cfg.WorkspaceID,
		EncryptedKey: resp.GetEncrypted(),
		CreatedAt:    time.Now().UnixMilli(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to insert acme user: %w", err)
	}

	config := lego.NewConfig(&user)
	client, err := lego.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create acme client: %w", err)
	}

	reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		return nil, fmt.Errorf("failed to register acme user: %w", err)
	}

	user.Registration = reg

	err = db.Query.UpdateAcmeUserRegistrationURI(ctx, cfg.DB.RW(), db.UpdateAcmeUserRegistrationURIParams{
		ID:              uint64(id),
		RegistrationUri: sql.NullString{Valid: true, String: reg.URI},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update acme user registration status: %w", err)
	}

	return client, nil
}
