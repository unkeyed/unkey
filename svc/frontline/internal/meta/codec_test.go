package meta

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/paseto"
)

const (
	testSigningKey      = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
	alternateSigningKey = "202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f"
)

func TestCodec_RoundTrip(t *testing.T) {
	t.Parallel()

	codec := newTestCodec(t)
	metadata := &Metadata{
		Claims: paseto.Claims{ExpiresAt: time.Unix(2_000_000_000, 0).UTC()},
		Hops: []Hop{
			{
				Region:        "aws::us-east-1",
				RequestID:     "req_123",
				FrontlineID:   "frontline_123",
				TimeUnixMilli: 1_777_000_000_123,
			},
		},
	}

	token, err := codec.Marshal(metadata)
	require.NoError(t, err)
	require.Regexp(t, `^v4\.public\.[A-Za-z0-9_-]+$`, token)

	opened, err := codec.Unmarshal(token)
	require.NoError(t, err)
	require.Equal(t, metadata, opened)
}

func TestCodec_DeterministicToken(t *testing.T) {
	t.Parallel()

	codec := newTestCodec(t)
	first, err := codec.Marshal(validMetadata())
	require.NoError(t, err)
	second, err := codec.Marshal(validMetadata())
	require.NoError(t, err)
	require.Equal(t, first, second)
}

func TestCodec_RejectsWrongSigningKey(t *testing.T) {
	t.Parallel()

	signer := newTestCodecWithSigningKey(t, testSigningKey)
	verifier := newTestCodecWithSigningKey(t, alternateSigningKey)
	token, err := signer.Marshal(validMetadata())
	require.NoError(t, err)

	_, err = verifier.Unmarshal(token)
	require.ErrorIs(t, err, paseto.ErrInvalidToken)
}

func TestCodec_RejectsMalformedToken(t *testing.T) {
	t.Parallel()

	codec := newTestCodec(t)
	tests := []struct {
		name  string
		token string
		err   string
	}{
		{name: "missing", token: "", err: "required"},
		{name: "one segment", token: "token", err: "three or four segments"},
		{name: "two segments", token: "v4.public", err: "three or four segments"},
		{name: "extra segment", token: "v4.public.payload.footer.extra", err: "three or four segments"},
		{name: "wrong purpose", token: "v4.local.YQ", err: "unexpected token header"},
		{name: "invalid payload encoding", token: "v4.public.%%%", err: "not canonical"},
		{name: "short payload", token: "v4.public.YQ", err: "too short"},
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
	token, err := codec.Marshal(validMetadata())
	require.NoError(t, err)
	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)

	body, err := base64.RawURLEncoding.DecodeString(parts[2])
	require.NoError(t, err)
	body[0] ^= 0xff
	payloadTampered := strings.Join([]string{parts[0], parts[1], base64.RawURLEncoding.EncodeToString(body)}, ".")
	_, err = codec.Unmarshal(payloadTampered)
	require.ErrorIs(t, err, paseto.ErrInvalidToken)

	body, err = base64.RawURLEncoding.DecodeString(parts[2])
	require.NoError(t, err)
	body[len(body)-1] ^= 0xff
	signatureTampered := strings.Join([]string{parts[0], parts[1], base64.RawURLEncoding.EncodeToString(body)}, ".")
	_, err = codec.Unmarshal(signatureTampered)
	require.ErrorIs(t, err, paseto.ErrInvalidToken)
}

// TestCodec_DoesNotValidateExpiration keeps token policy at the request
// boundary instead of the metadata codec.
func TestCodec_DoesNotValidateExpiration(t *testing.T) {
	t.Parallel()

	codec := newTestCodec(t)
	metadata := validMetadata()
	metadata.ExpiresAt = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
	token, err := codec.Marshal(metadata)
	require.NoError(t, err)

	opened, err := codec.Unmarshal(token)
	require.NoError(t, err)
	require.Equal(t, metadata, opened)
}

func TestCodec_RejectsOversizedMetadata(t *testing.T) {
	t.Parallel()

	codec := newTestCodec(t)
	metadata := validMetadata()
	metadata.Subject = strings.Repeat("a", maxTokenBytes)

	token, err := codec.Marshal(metadata)
	require.ErrorContains(t, err, "encoded metadata exceeds")
	require.Empty(t, token)
}

func TestCodec_RejectsOversizedSignedToken(t *testing.T) {
	t.Parallel()

	codec := newTestCodec(t)
	metadata := validMetadata()
	metadata.Subject = strings.Repeat("a", maxTokenBytes)
	token, err := codec.signer.Sign(paseto.Message[Metadata]{
		Payload: *metadata,
		Footer:  nil,
	})
	require.NoError(t, err)
	require.Greater(t, len(token), maxTokenBytes)

	_, err = codec.Unmarshal(token)
	require.ErrorContains(t, err, "encoded metadata exceeds")
}

func TestCodec_DoesNotValidateMetadataFields(t *testing.T) {
	t.Parallel()

	codec := newTestCodec(t)
	metadata := &Metadata{
		Hops: []Hop{{TimeUnixMilli: -1}},
	}
	token, err := codec.Marshal(metadata)
	require.NoError(t, err)
	opened, err := codec.Unmarshal(token)
	require.NoError(t, err)
	require.Equal(t, metadata, opened)
}

func TestCodec_UnmarshalCanReadTokenMoreThanOnce(t *testing.T) {
	t.Parallel()

	codec := newTestCodec(t)
	token, err := codec.Marshal(validMetadata())
	require.NoError(t, err)

	first, err := codec.Unmarshal(token)
	require.NoError(t, err)
	second, err := codec.Unmarshal(token)
	require.NoError(t, err)
	require.Equal(t, first, second)
}

func TestCodec_MarshalReflectsMetadata(t *testing.T) {
	t.Parallel()

	codec := newTestCodec(t)
	first, err := codec.Marshal(validMetadata())
	require.NoError(t, err)

	metadata := validMetadata()
	metadata.Hops = append(metadata.Hops, Hop{Region: "aws::eu-west-1"})
	second, err := codec.Marshal(metadata)
	require.NoError(t, err)
	require.NotEqual(t, first, second)
	opened, err := codec.Unmarshal(second)
	require.NoError(t, err)
	require.Equal(t, metadata.Hops, opened.Hops)
}

func TestCodec_MarshalRequiresMetadata(t *testing.T) {
	t.Parallel()

	codec := newTestCodec(t)
	token, err := codec.Marshal(nil)
	require.ErrorContains(t, err, "metadata is required")
	require.Empty(t, token)
}

func TestNew_RequiresHexEncodedEd25519Seed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		signingKey string
		err        string
	}{
		{name: "empty", signingKey: "", err: "64 hexadecimal characters"},
		{name: "short", signingKey: strings.Repeat("00", 31), err: "64 hexadecimal characters"},
		{name: "invalid hexadecimal", signingKey: strings.Repeat("z", 64), err: "decode metadata signing key"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			codec, err := New(test.signingKey)
			require.ErrorContains(t, err, test.err)
			require.Nil(t, codec)
		})
	}
}

func newTestCodec(t *testing.T) *Codec {
	t.Helper()
	return newTestCodecWithSigningKey(t, testSigningKey)
}

func newTestCodecWithSigningKey(t *testing.T, signingKey string) *Codec {
	t.Helper()
	codec, err := New(signingKey)
	require.NoError(t, err)
	return codec
}

func validMetadata() *Metadata {
	return &Metadata{
		Claims: paseto.Claims{ExpiresAt: time.Unix(2_000_000_000, 0).UTC()},
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
