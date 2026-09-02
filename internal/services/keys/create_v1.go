package keys

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"hash/crc32"
	"io"
	"strings"

	"github.com/unkeyed/unkey/pkg/base58"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/hash"
)

// Version 1 fields use fixed widths so secret scanners can match keys without parsing them.
const (
	keyV1Base                 = len(base58.Alphabet)
	keyV1ByteValueCount       = 1 << 8
	keyV1Marker               = "unkeyv1"
	keyV1RandomBatchBytes     = 64
	keyV1RandomCandidateLimit = byte(keyV1ByteValueCount / keyV1Base * keyV1Base)
	keyV1RandomLength         = 44
	keyV1RandomHead           = 8
	keyV1ChecksumLength       = 6
)

var keyV1ChecksumTable = crc32.MakeTable(crc32.Castagnoli)

// CreateKeyV1 generates an API key in the version 1 plaintext format. A non-empty
// prefix must match ^[A-Za-z0-9_]{0,15}[A-Za-z0-9]$.
func (s *service) CreateKeyV1(_ context.Context, req CreateKeyV1Request) (CreateKeyV1Response, error) {
	// Direct byte checks take about 34 ns for a valid 16-character prefix. A
	// precompiled regex takes about 301 ns.
	validPrefix := len(req.Prefix) <= 16
	if validPrefix {
		for i := 0; i < len(req.Prefix); i++ {
			character := req.Prefix[i]
			isLetter := character >= 'A' && character <= 'Z' || character >= 'a' && character <= 'z'
			isNumber := character >= '0' && character <= '9'
			if isLetter || isNumber {
				continue
			}
			if character == '_' && i < len(req.Prefix)-1 {
				continue
			}

			validPrefix = false
			break
		}
	}

	if !validPrefix {
		return CreateKeyV1Response{}, fault.New(
			"invalid API key prefix",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("prefix must be empty or match ^[A-Za-z0-9_]{0,15}[A-Za-z0-9]$"),
			fault.Public("The prefix must be empty or contain 1 to 16 ASCII letters, numbers, or underscores. A non-empty prefix must end with a letter or number."),
		)
	}

	random, err := generateKeyV1Random(rand.Reader)
	if err != nil {
		return CreateKeyV1Response{}, fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to generate random key characters"),
			fault.Public("Failed to generate a secure key."),
		)
	}

	return formatKeyV1(req.Prefix, random), nil
}

// generateKeyV1Random creates 44 random Base58 characters. Every character has
// the same chance of appearing.
//
// A byte has 256 possible values, but the Base58 alphabet has 58 characters. If
// we used every byte value, the first 24 characters would appear more often. We
// use values 0 through 231 so each character gets exactly four byte values. We
// ignore values 232 through 255.
//
// Read 64 bytes at a time. One batch provides at least 44 usable bytes about
// 99.99998 percent of the time. If it does not, keep the generated characters
// and read another batch.
func generateKeyV1Random(reader io.Reader) ([keyV1RandomLength]byte, error) {
	random := [keyV1RandomLength]byte{}
	written := 0

	for written < len(random) {
		candidates := [keyV1RandomBatchBytes]byte{}
		_, err := io.ReadFull(reader, candidates[:])
		if err != nil {
			return random, err
		}

		for _, candidate := range candidates {
			if candidate >= keyV1RandomCandidateLimit {
				continue
			}

			random[written] = base58.Alphabet[candidate%byte(keyV1Base)]
			written++
			if written == len(random) {
				break
			}
		}
	}

	return random, nil
}

// formatKeyV1 assembles a version 1 plaintext key and its storage metadata.
func formatKeyV1(prefix string, random [keyV1RandomLength]byte) CreateKeyV1Response {
	randomText := string(random[:])

	unsignedKey := randomText[:keyV1RandomHead] + keyV1Marker + randomText[keyV1RandomHead:]
	if prefix != "" {
		unsignedKey = prefix + "_" + unsignedKey
	}
	checksumBytes := [4]byte{}
	binary.BigEndian.PutUint32(checksumBytes[:], crc32.Checksum([]byte(unsignedKey), keyV1ChecksumTable))
	checksum := base58.Encode(checksumBytes[:])
	checksum = strings.Repeat("1", keyV1ChecksumLength-len(checksum)) + checksum
	key := unsignedKey + checksum

	return CreateKeyV1Response{
		Key:    key,
		Hash:   hash.Sha256(key),
		Prefix: prefix,
		Start:  randomText[:4],
		End:    key[len(key)-4:],
	}
}
