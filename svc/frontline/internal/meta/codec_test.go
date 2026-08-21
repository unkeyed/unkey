package meta

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/jwt"
)

func TestCodec_RoundTrip(t *testing.T) {
	t.Parallel()

	codec := newTestCodec(t)
	payload := &Metadata{
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: 2_000_000_000},
		Hops: []Hop{
			{
				Region:        "aws::us-east-1",
				RequestID:     "req_123",
				FrontlineID:   "frontline_123",
				TimeUnixMilli: 1_777_000_000_123,
			},
		},
	}

	token, err := codec.Marshal(payload)
	require.NoError(t, err)
	require.Regexp(t, `^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$`, token)

	opened, err := codec.Unmarshal(token)
	require.NoError(t, err)
	require.Equal(t, payload, opened)
}

func TestCodec_DeterministicToken(t *testing.T) {
	t.Parallel()

	codec := newTestCodec(t)
	first, err := codec.Marshal(validPayload())
	require.NoError(t, err)
	second, err := codec.Marshal(validPayload())
	require.NoError(t, err)
	require.Equal(t, first, second)
}

func TestCodec_RejectsWrongSigningKey(t *testing.T) {
	t.Parallel()

	signer := newTestCodecWithSigningKey(t, "signing-key-a")
	verifier := newTestCodecWithSigningKey(t, "signing-key-b")
	token, err := signer.Marshal(validPayload())
	require.NoError(t, err)

	_, err = verifier.Unmarshal(token)
	require.ErrorContains(t, err, "invalid signature")
}

func TestCodec_RejectsMalformedToken(t *testing.T) {
	t.Parallel()

	codec := newTestCodec(t)
	const header = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"
	const payload = "e30"
	tests := []struct {
		name  string
		token string
		err   string
	}{
		{name: "missing", token: "", err: "required"},
		{name: "one part", token: "token", err: "3 parts"},
		{name: "two parts", token: "header.payload", err: "3 parts"},
		{name: "extra part", token: "header.payload.signature.extra", err: "3 parts"},
		{name: "invalid header base64", token: "%%%." + payload + ".signature", err: "invalid header encoding"},
		{name: "invalid header json", token: "bm90LWpzb24." + payload + ".signature", err: "invalid header JSON"},
		{name: "invalid signature base64", token: header + "." + payload + ".%%%", err: "invalid signature encoding"},
		{name: "short signature", token: header + "." + payload + ".YQ", err: "invalid signature"},
		{name: "too large", token: strings.Repeat("a", maxTokenBytes+1), err: "exceeds"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := codec.Unmarshal(test.token)
			require.ErrorContains(t, err, test.err)
		})
	}
}

func TestCodec_RejectsTampering(t *testing.T) {
	t.Parallel()

	codec := newTestCodec(t)
	token, err := codec.Marshal(validPayload())
	require.NoError(t, err)
	parts := strings.Split(token, ".")

	payloadTampered := strings.Join([]string{
		parts[0],
		base64.RawURLEncoding.EncodeToString([]byte(`{"hops":[{"region":"aws::eu-west-1"}]}`)),
		parts[2],
	}, ".")
	_, err = codec.Unmarshal(payloadTampered)
	require.ErrorContains(t, err, "invalid signature")

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	require.NoError(t, err)
	signature[0] ^= 0xff
	signatureTampered := strings.Join([]string{
		parts[0],
		parts[1],
		base64.RawURLEncoding.EncodeToString(signature),
	}, ".")
	_, err = codec.Unmarshal(signatureTampered)
	require.ErrorContains(t, err, "invalid signature")
}

func TestCodec_RejectsExpiredToken(t *testing.T) {
	t.Parallel()

	codec := newTestCodec(t)
	payload := validPayload()
	payload.ExpiresAt = time.Now().Add(-time.Second).Unix()
	token, err := codec.Marshal(payload)
	require.NoError(t, err)

	_, err = codec.Unmarshal(token)
	require.ErrorIs(t, err, jwt.ErrTokenExpired)
}

func TestCodec_RejectsOversizedPayload(t *testing.T) {
	t.Parallel()

	codec := newTestCodec(t)
	payload := validPayload()
	payload.Subject = strings.Repeat("a", maxTokenBytes)

	token, err := codec.Marshal(payload)
	require.ErrorContains(t, err, "encoded metadata exceeds")
	require.Empty(t, token)
}

func TestCodec_RejectsOversizedSignedToken(t *testing.T) {
	t.Parallel()

	codec := newTestCodec(t)
	signer, err := jwt.NewHS256Signer[map[string]string]([]byte("shared-signing-key"))
	require.NoError(t, err)
	token, err := signer.Sign(map[string]string{
		"data": strings.Repeat("a", maxTokenBytes),
	})
	require.NoError(t, err)
	require.Greater(t, len(token), maxTokenBytes)

	_, err = codec.Unmarshal(token)
	require.ErrorContains(t, err, "encoded metadata exceeds")
}

func TestCodec_DoesNotValidateMetadataFields(t *testing.T) {
	t.Parallel()

	codec := newTestCodec(t)
	payload := &Metadata{
		Hops: []Hop{{TimeUnixMilli: -1}},
	}
	token, err := codec.Marshal(payload)
	require.NoError(t, err)
	opened, err := codec.Unmarshal(token)
	require.NoError(t, err)
	require.Equal(t, payload, opened)
}

func TestCodec_UnmarshalCanReadTokenMoreThanOnce(t *testing.T) {
	t.Parallel()

	codec := newTestCodec(t)
	token, err := codec.Marshal(validPayload())
	require.NoError(t, err)

	first, err := codec.Unmarshal(token)
	require.NoError(t, err)
	second, err := codec.Unmarshal(token)
	require.NoError(t, err)
	require.Equal(t, first, second)
}

func TestCodec_MarshalReflectsPayload(t *testing.T) {
	t.Parallel()

	codec := newTestCodec(t)
	first, err := codec.Marshal(validPayload())
	require.NoError(t, err)

	payload := validPayload()
	payload.Hops = append(payload.Hops, Hop{Region: "aws::eu-west-1"})
	second, err := codec.Marshal(payload)
	require.NoError(t, err)
	require.NotEqual(t, first, second)
	opened, err := codec.Unmarshal(second)
	require.NoError(t, err)
	require.Equal(t, payload.Hops, opened.Hops)
}

func TestCodec_MarshalRequiresPayload(t *testing.T) {
	t.Parallel()

	codec := newTestCodec(t)
	token, err := codec.Marshal(nil)
	require.ErrorContains(t, err, "payload is required")
	require.Empty(t, token)
}

func TestNew_RequiresSigningKey(t *testing.T) {
	t.Parallel()

	codec, err := New("")
	require.ErrorContains(t, err, "secret must not be empty")
	require.Nil(t, codec)
}

func newTestCodec(t *testing.T) *Codec {
	t.Helper()
	return newTestCodecWithSigningKey(t, "shared-signing-key")
}

func newTestCodecWithSigningKey(t *testing.T, signingKey string) *Codec {
	t.Helper()
	codec, err := New(signingKey)
	require.NoError(t, err)
	return codec
}

func validPayload() *Metadata {
	return &Metadata{
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: 2_000_000_000},
		Hops: []Hop{
			{
				Region:        "aws::us-east-1",
				RequestID:     "req_123",
				FrontlineID:   "frontline_123",
				TimeUnixMilli: 1_777_000_000_123,
			},
		},
	}
}
