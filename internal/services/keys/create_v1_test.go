package keys

import (
	"bytes"
	"context"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/base58"
	"github.com/unkeyed/unkey/pkg/hash"
)

// TestCreateKeyV1_Format guarantees that generated keys match the version 1 grammar.
func TestCreateKeyV1_Format(t *testing.T) {
	t.Parallel()

	service := &service{}
	response, err := service.CreateKeyV1(context.Background(), CreateKeyV1Request{Prefix: "prod_sk"})
	require.NoError(t, err)

	pattern := regexp.MustCompile(`^prod_sk_[1-9A-HJ-NP-Za-km-z]{8}unkeyv1[1-9A-HJ-NP-Za-km-z]{42}$`)
	require.Regexp(t, pattern, response.Key)
	require.Len(t, response.Key, 65)
	require.Equal(t, "prod_sk", response.Prefix)
	require.Equal(t, response.Key[len(response.Prefix)+1:len(response.Prefix)+5], response.Start)
	require.Equal(t, response.Key[len(response.Key)-4:], response.End)
	require.Equal(t, hash.Sha256(response.Key), response.Hash)
}

func TestCreateKeyV1_PrefixValidation(t *testing.T) {
	t.Parallel()

	service := &service{}
	testCases := []struct {
		name      string
		prefix    string
		wantValid bool
	}{
		{name: "one character", prefix: "a", wantValid: true},
		{name: "embedded underscores", prefix: "prod_sk", wantValid: true},
		{name: "empty", prefix: "", wantValid: false},
		{name: "ends with underscore", prefix: "prod_", wantValid: false},
		{name: "eight characters", prefix: "prod_key", wantValid: false},
		{name: "hyphen", prefix: "prod-sk", wantValid: false},
		{name: "non ASCII", prefix: "prød", wantValid: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			response, err := service.CreateKeyV1(
				context.Background(),
				CreateKeyV1Request{Prefix: testCase.prefix},
			)
			if testCase.wantValid {
				require.NoError(t, err)
				require.Len(t, response.Key, len(testCase.prefix)+58)
				return
			}

			require.Error(t, err)
			require.Empty(t, response)
		})
	}
}

// TestGenerateKeyV1Random_ByteMapping guarantees that ignored bytes do not enter
// the output and that the generator reads another batch when necessary.
func TestGenerateKeyV1Random_ByteMapping(t *testing.T) {
	t.Parallel()

	t.Run("ignored values do not enter output", func(t *testing.T) {
		t.Parallel()

		candidates := make([]byte, keyV1RandomBatchBytes)
		candidates[0] = keyV1RandomCandidateLimit
		for i := range keyV1RandomLength {
			candidates[i+1] = byte(i)
		}

		random, err := generateKeyV1Random(bytes.NewReader(candidates))
		require.NoError(t, err)
		require.Equal(t, base58.Alphabet[:keyV1RandomLength], string(random[:]))
	})

	t.Run("insufficient first batch reads another", func(t *testing.T) {
		t.Parallel()

		ignored := bytes.Repeat([]byte{keyV1RandomCandidateLimit}, keyV1RandomBatchBytes)
		accepted := make([]byte, keyV1RandomBatchBytes)
		for i := range keyV1RandomLength {
			accepted[i] = byte(i)
		}
		candidates := append(ignored, accepted...)

		random, err := generateKeyV1Random(bytes.NewReader(candidates))
		require.NoError(t, err)
		require.Equal(t, base58.Alphabet[:keyV1RandomLength], string(random[:]))
	})
}

func TestGenerateKeyV1Random_ReaderFailure(t *testing.T) {
	t.Parallel()

	_, err := generateKeyV1Random(bytes.NewReader(nil))
	require.Error(t, err)
}

// TestFormatKeyV1_KnownVector guards the marker position and CRC-32C checksum.
func TestFormatKeyV1_KnownVector(t *testing.T) {
	t.Parallel()

	random := [keyV1RandomLength]byte{}
	copy(random[:], "111thX6LZfHDZZKUs92febYZhYRcXddmzfzF2NvTkPNE")

	response := formatKeyV1("unkey", random)
	require.Equal(
		t,
		"unkey_111thX6Lunkeyv1ZfHDZZKUs92febYZhYRcXddmzfzF2NvTkPNE18n8B8",
		response.Key,
	)
}
