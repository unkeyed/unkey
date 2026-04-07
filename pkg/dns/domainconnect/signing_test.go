package domainconnect

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func generateTestKey(t *testing.T) ([]byte, *rsa.PublicKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: func() []byte { b, _ := x509.MarshalPKCS8PrivateKey(key); return b }(),
	})
	return pemBytes, &key.PublicKey
}

// TestSignSyncURL_ExcludesSigAndKey verifies that the signature is computed
// over the full query string excluding only "sig" and "key" per the Domain
// Connect spec: "The digital signature will be generated on the full query
// string only, excluding the sig and key parameters."
func TestSignSyncURL_ExcludesSigAndKey(t *testing.T) {
	t.Parallel()
	pemBytes, pubKey := generateTestKey(t)

	rawURL := "https://example.com/apply?domain=example.com&target=abc.unkey-dns.com&redirect_uri=https%3A%2F%2Fapp.unkey.com%2Fsettings"

	signed, err := signSyncURL(rawURL, pemBytes, "_dcpubkeyv1")
	require.NoError(t, err)

	u, err := url.Parse(signed)
	require.NoError(t, err)
	q := u.Query()

	// key and sig must be present in the final URL.
	require.Equal(t, "_dcpubkeyv1", q.Get("key"))
	require.NotEmpty(t, q.Get("sig"))

	// Reconstruct what should have been signed: all params except sig and key.
	signParams := url.Values{}
	for k, v := range q {
		if k != "sig" && k != "key" {
			signParams[k] = v
		}
	}
	dataToSign := signParams.Encode()

	// Verify the signature with the public key.
	sigBytes, err := base64.StdEncoding.DecodeString(q.Get("sig"))
	require.NoError(t, err)

	hash := sha256.Sum256([]byte(dataToSign))
	err = rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, hash[:], sigBytes)
	require.NoError(t, err, "signature must verify against query string excluding sig and key")
}

// TestSignSyncURL_KeyNotInSignedData ensures that including "key" in the
// verification data causes failure — proving our signature excludes it.
func TestSignSyncURL_KeyNotInSignedData(t *testing.T) {
	t.Parallel()
	pemBytes, pubKey := generateTestKey(t)

	rawURL := "https://example.com/apply?domain=example.com&target=abc.unkey-dns.com"

	signed, err := signSyncURL(rawURL, pemBytes, "_dcpubkeyv1")
	require.NoError(t, err)

	u, err := url.Parse(signed)
	require.NoError(t, err)
	q := u.Query()

	// If we include "key" in the data (the old bug), verification must fail.
	wrongParams := url.Values{}
	for k, v := range q {
		if k != "sig" {
			wrongParams[k] = v // includes "key" — wrong per spec
		}
	}
	wrongData := wrongParams.Encode()

	sigBytes, err := base64.StdEncoding.DecodeString(q.Get("sig"))
	require.NoError(t, err)

	hash := sha256.Sum256([]byte(wrongData))
	err = rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, hash[:], sigBytes)
	require.Error(t, err, "signature must NOT verify when key is included in signed data")
}

// TestSignSyncURL_SigIsLastParam verifies that "sig" is the last query
// parameter in the URL (Cloudflare requirement).
func TestSignSyncURL_SigIsLastParam(t *testing.T) {
	t.Parallel()
	pemBytes, _ := generateTestKey(t)

	rawURL := "https://example.com/apply?domain=example.com&z=last&a=first"

	signed, err := signSyncURL(rawURL, pemBytes, "_dcpubkeyv1")
	require.NoError(t, err)

	u, err := url.Parse(signed)
	require.NoError(t, err)

	// sig must be the last parameter in the raw query string.
	raw := u.RawQuery
	lastAmp := lastIndexByte(raw, '&')
	require.Greater(t, lastAmp, 0)
	require.True(t, raw[lastAmp+1:lastAmp+5] == "sig=", "last query parameter must be sig, got: %s", raw[lastAmp+1:])
}

// TestSignSyncURL_AllParamsIncluded verifies that redirect_uri, state, and
// other non-template params are included in the signature — not just domain
// and host.
func TestSignSyncURL_AllParamsIncluded(t *testing.T) {
	t.Parallel()
	pemBytes, pubKey := generateTestKey(t)

	rawURL := "https://example.com/apply?domain=example.com&host=sub&target=t&redirect_uri=https%3A%2F%2Fapp.unkey.com&state=csrf123&force=1"

	signed, err := signSyncURL(rawURL, pemBytes, "_dcpubkeyv1")
	require.NoError(t, err)

	u, err := url.Parse(signed)
	require.NoError(t, err)
	q := u.Query()

	// All original params must be present in the URL.
	require.Equal(t, "example.com", q.Get("domain"))
	require.Equal(t, "sub", q.Get("host"))
	require.Equal(t, "t", q.Get("target"))
	require.Equal(t, "https://app.unkey.com", q.Get("redirect_uri"))
	require.Equal(t, "csrf123", q.Get("state"))
	require.Equal(t, "1", q.Get("force"))

	// Verify against full query string minus sig and key.
	signParams := url.Values{}
	for k, v := range q {
		if k != "sig" && k != "key" {
			signParams[k] = v
		}
	}

	sigBytes, err := base64.StdEncoding.DecodeString(q.Get("sig"))
	require.NoError(t, err)

	hash := sha256.Sum256([]byte(signParams.Encode()))
	err = rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, hash[:], sigBytes)
	require.NoError(t, err)
}

// TestSignSyncURL_URLEncodedBeforeSigning verifies that query param values are
// URL-encoded in the signed data, per spec: "The values of each query string
// value key/value pair must be properly URL Encoded before the signature is
// generated."
func TestSignSyncURL_URLEncodedBeforeSigning(t *testing.T) {
	t.Parallel()
	pemBytes, pubKey := generateTestKey(t)

	// redirect_uri contains characters that must be URL-encoded.
	rawURL := "https://example.com/apply?domain=example.com&redirect_uri=https%3A%2F%2Fapp.unkey.com%2Fws%2Fprojects%2Fp1%2Fsettings"

	signed, err := signSyncURL(rawURL, pemBytes, "_dcpubkeyv1")
	require.NoError(t, err)

	u, err := url.Parse(signed)
	require.NoError(t, err)
	q := u.Query()

	// The signed data should use url.Values.Encode() which URL-encodes values.
	signParams := url.Values{}
	signParams.Set("domain", "example.com")
	signParams.Set("redirect_uri", "https://app.unkey.com/ws/projects/p1/settings")
	expectedData := signParams.Encode()

	// Verify the signature matches URL-encoded data.
	sigBytes, err := base64.StdEncoding.DecodeString(q.Get("sig"))
	require.NoError(t, err)

	hash := sha256.Sum256([]byte(expectedData))
	err = rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, hash[:], sigBytes)
	require.NoError(t, err, "signature must be computed over URL-encoded values")
}

// TestSignSyncURL_PKCS1Key verifies that PKCS1-formatted keys work.
func TestSignSyncURL_PKCS1Key(t *testing.T) {
	t.Parallel()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	rawURL := "https://example.com/apply?domain=example.com"
	signed, err := signSyncURL(rawURL, pemBytes, "_dcpubkeyv1")
	require.NoError(t, err)

	u, err := url.Parse(signed)
	require.NoError(t, err)

	sigBytes, err := base64.StdEncoding.DecodeString(u.Query().Get("sig"))
	require.NoError(t, err)

	signParams := url.Values{}
	signParams.Set("domain", "example.com")
	hash := sha256.Sum256([]byte(signParams.Encode()))
	err = rsa.VerifyPKCS1v15(&key.PublicKey, crypto.SHA256, hash[:], sigBytes)
	require.NoError(t, err)
}

// TestSignSyncURL_NoSigtsParam verifies we don't add the non-spec "sigts"
// parameter that the Railway library added.
func TestSignSyncURL_NoSigtsParam(t *testing.T) {
	t.Parallel()
	pemBytes, _ := generateTestKey(t)

	rawURL := "https://example.com/apply?domain=example.com"
	signed, err := signSyncURL(rawURL, pemBytes, "_dcpubkeyv1")
	require.NoError(t, err)

	u, err := url.Parse(signed)
	require.NoError(t, err)

	require.Empty(t, u.Query().Get("sigts"), "sigts is not part of the Domain Connect spec and must not be added")
}

// TestSignSyncURL_Base64StdEncoding verifies the signature uses standard
// base64 encoding (not URL-safe or raw).
func TestSignSyncURL_Base64StdEncoding(t *testing.T) {
	t.Parallel()
	pemBytes, _ := generateTestKey(t)

	rawURL := "https://example.com/apply?domain=example.com"
	signed, err := signSyncURL(rawURL, pemBytes, "_dcpubkeyv1")
	require.NoError(t, err)

	u, err := url.Parse(signed)
	require.NoError(t, err)

	sig := u.Query().Get("sig")
	require.NotEmpty(t, sig)

	// Must decode with StdEncoding (with padding).
	_, err = base64.StdEncoding.DecodeString(sig)
	require.NoError(t, err, "signature must be valid standard base64")

	// Length must be padded to multiple of 4 (StdEncoding uses = padding).
	require.Equal(t, 0, len(sig)%4, "standard base64 must be padded to multiple of 4")
}

func lastIndexByte(s string, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			return i
		}
	}
	return -1
}
