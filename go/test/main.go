package main

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge/http01"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
	vaultv1 "github.com/unkeyed/unkey/go/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/vault"
	"github.com/unkeyed/unkey/go/pkg/vault/storage"
)

// You'll need a user or account type that implements acme.User
type AcmeUser struct {
	WorkspaceID  string
	Registration *registration.Resource
	key          crypto.PrivateKey
}

func (u *AcmeUser) GetEmail() string {
	return fmt.Sprintf("workspace-%s@steamsets.com", u.WorkspaceID)
}

func (u AcmeUser) GetRegistration() *registration.Resource {
	return u.Registration
}

func (u *AcmeUser) GetPrivateKey() crypto.PrivateKey {
	return u.key
}

func main() {
	ctx := context.Background()
	logger := logging.New()

	vaultStorage, err := storage.NewS3(storage.S3Config{
		Logger:            logger,
		S3URL:             "http://127.0.0.1:3902",
		S3Bucket:          "vault",
		S3AccessKeyID:     "minio_root_user",
		S3AccessKeySecret: "minio_root_password",
	})
	if err != nil {
		panic(fmt.Errorf("unable to create vault storage: %w", err))
	}

	vaultSvc, err := vault.New(vault.Config{
		Logger:     logger,
		Storage:    vaultStorage,
		MasterKeys: []string{"Ch9rZWtfMmdqMFBJdVhac1NSa0ZhNE5mOWlLSnBHenFPENTt7an5MRogENt9Si6wms4pQ2XIvqNSIgNpaBenJmXgcInhu6Nfv2U="},
	})
	if err != nil {
		panic(fmt.Errorf("unable to create vault service: %w", err))
	}

	DB, err := db.New(db.Config{
		PrimaryDSN: "unkey:password@tcp(127.0.0.1:3306)/unkey?parseTime=true&interpolateParams=true",
		Logger:     logger,
	})
	if err != nil {
		panic(fmt.Errorf("unable to create database: %w", err))
	}

	user, err := db.Query.FindAcmeUserByWorkspaceID(ctx, DB.RO(), "unkey")
	if err != nil && !db.IsNotFound(err) {
		panic(fmt.Errorf("unable to find user: %w", err))
	}

	var acmeUser AcmeUser
	register := db.IsNotFound(err)
	if db.IsNotFound(err) {
		privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			panic(fmt.Errorf("unable to generate private key: %w", err))
		}

		privateKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
		if err != nil {
			panic(fmt.Errorf("unable to marshal private key: %w", err))
		}

		privateKeyPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "EC PRIVATE KEY",
			Bytes: privateKeyBytes,
		})

		res, err := vaultSvc.Encrypt(ctx, &vaultv1.EncryptRequest{
			Data:    string(privateKeyPEM),
			Keyring: "unkey",
		})
		if err != nil {
			panic(fmt.Errorf("unable to encrypt private key: %w", err))
		}

		log.Printf("encrypted private key: %s", res.Encrypted)

		acmeUser = AcmeUser{
			WorkspaceID: "unkey",
			key:         privateKey,
		}
	} else {
		res, err := vaultSvc.Decrypt(ctx, &vaultv1.DecryptRequest{
			Encrypted: user.EncryptedKey,
			Keyring:   user.WorkspaceID,
		})
		if err != nil {
			panic(fmt.Errorf("unable to decrypt private key: %w", err))
		}

		block, _ := pem.Decode([]byte(res.Plaintext))
		privateKey, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			panic(fmt.Errorf("unable to parse private key: %w", err))
		}

		acmeUser = AcmeUser{
			key:         privateKey,
			WorkspaceID: user.WorkspaceID,
		}
	}

	config := lego.NewConfig(&acmeUser)
	client, err := lego.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}

	if register {
		reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
		if err != nil {
			log.Fatal(err)
		}

		acmeUser.Registration = reg
	}

	// We specify an HTTP port of 5002 and an TLS port of 5001 on all interfaces
	// because we aren't running as root and can't bind a listener to port 80 and 443
	// (used later when we attempt to pass challenges). Keep in mind that you still
	// need to proxy challenge traffic to port 5002 and 5001.
	err = client.Challenge.SetHTTP01Provider(http01.NewProviderServer("", "5002"))
	if err != nil {
		log.Fatal(err)
	}

	certificates, err := client.Certificate.Obtain(certificate.ObtainRequest{
		Domains: []string{"fun.steamsets.dev"},
		Bundle:  true,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Each certificate comes back with the cert bytes, the bytes of the client's
	// private key, and a certificate URL. SAVE THESE TO DISK.
	fmt.Printf("%#v\n", certificates)

	// ... all done.
}
