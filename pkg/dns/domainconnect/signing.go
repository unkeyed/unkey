package domainconnect

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/url"
)

// signSyncURL signs a Domain Connect sync URL per spec section 5.2.1:
//
//	"The digital signature will be generated on the full query string only,
//	 excluding the sig and key parameters."
//
// The data-to-sign is built by taking all query parameters except "sig" and "key",
// sorting them alphabetically by key, URL-encoding values, and joining with "&"
// (this is what url.Values.Encode does). The result is hashed with SHA-256 and
// signed with RSA PKCS1v15.
//
// The signature is base64 standard-encoded and appended as the last query parameter.
// Appending last is required by Cloudflare's implementation.
func signSyncURL(rawURL string, privateKeyPEM []byte, keyID string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse URL: %w", err)
	}

	q := u.Query()
	q.Set("key", keyID)

	// Build data to sign: all query params except "sig" and "key", sorted and URL-encoded.
	signParams := url.Values{}
	for k, v := range q {
		if k != "sig" && k != "key" {
			signParams[k] = v
		}
	}
	dataToSign := signParams.Encode()

	rsaKey, err := parseRSAPrivateKey(privateKeyPEM)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256([]byte(dataToSign))
	sigBytes, err := rsa.SignPKCS1v15(rand.Reader, rsaKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", fmt.Errorf("sign: %w", err)
	}

	sig := base64.StdEncoding.EncodeToString(sigBytes)

	// sig must be appended last (Cloudflare requirement).
	u.RawQuery = q.Encode() + "&sig=" + url.QueryEscape(sig)

	return u.String(), nil
}

// parseRSAPrivateKey decodes PEM and parses as RSA private key (PKCS8 or PKCS1).
func parseRSAPrivateKey(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		k, err2 := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("parse private key (tried PKCS8 and PKCS1): %w", err2)
		}
		return k, nil
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("expected RSA private key")
	}
	return rsaKey, nil
}
