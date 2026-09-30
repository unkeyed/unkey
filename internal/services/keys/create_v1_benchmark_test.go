package keys

import (
	"crypto/rand"
	"io"
	"math/big"
	"regexp"
	"testing"

	"github.com/unkeyed/unkey/pkg/base58"
)

// keyV1RandomRangeSampleBytesBenchmark is the input width for fair number mapping.
const keyV1RandomRangeSampleBytesBenchmark = 33

// benchmarkKeyV1RandomSink prevents the compiler from removing benchmark work.
var benchmarkKeyV1RandomSink [keyV1RandomLength]byte

// benchmarkKeyV1PrefixSink prevents the compiler from removing benchmark work.
var benchmarkKeyV1PrefixSink bool

// keyV1RandomRangeSpaceBenchmark is the number of possible 44-character Base58 strings.
var keyV1RandomRangeSpaceBenchmark = new(big.Int).Exp(
	big.NewInt(int64(keyV1Base)),
	big.NewInt(keyV1RandomLength),
	nil,
)

// keyV1RandomRangeLimitBenchmark excludes the incomplete range that would bias number mapping.
var keyV1RandomRangeLimitBenchmark = func() *big.Int {
	sampleSpace := new(big.Int).Lsh(big.NewInt(1), keyV1RandomRangeSampleBytesBenchmark*8)
	completeRanges := new(big.Int).Div(sampleSpace, keyV1RandomRangeSpaceBenchmark)
	return new(big.Int).Mul(completeRanges, keyV1RandomRangeSpaceBenchmark)
}()

// BenchmarkGenerateKeyV1Random compares five ways to produce a 44-character random field:
//
//   - 33-byte fair number mapping reads one large number. It retries when mapping
//     that number would make some outputs more likely than others.
//   - 64-byte fair character mapping ignores byte values from 232 through 255.
//     It maps each remaining byte to one Base58 character.
//   - 32-byte encoding converts 32 bytes to Base58 and adds leading "1"
//     characters until the output has 44 characters. It cannot produce every
//     possible 44-character Base58 string.
//   - 48-byte encoding converts 48 bytes to Base58 and discards characters from
//     the beginning until 44 remain. Some outputs are more likely than others.
//   - 64-byte encoding converts 64 bytes to Base58 and discards characters from
//     the beginning until 44 remain. Some outputs are more likely than others.
func BenchmarkGenerateKeyV1Random(b *testing.B) {
	benchmarks := []struct {
		name     string
		generate func(io.Reader) ([keyV1RandomLength]byte, error)
	}{
		{name: "fair_33_byte_number_mapping", generate: generateKeyV1RandomRange33Benchmark},
		{name: "fair_64_byte_character_mapping", generate: generateKeyV1Random},
		{name: "biased_32_byte_encoding_with_leading_ones", generate: generateKeyV1RandomFixed32Benchmark},
		{name: "biased_48_byte_encoding_keep_last_44", generate: generateKeyV1RandomTruncated48Benchmark},
		{name: "biased_64_byte_encoding_keep_last_44", generate: generateKeyV1RandomTruncated64Benchmark},
	}

	for _, benchmark := range benchmarks {
		b.Run(benchmark.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				random, err := benchmark.generate(rand.Reader)
				if err != nil {
					b.Fatal(err)
				}
				benchmarkKeyV1RandomSink = random
			}
		})
	}
}

// BenchmarkValidateKeyV1Prefix compares direct byte checks with a precompiled regex.
func BenchmarkValidateKeyV1Prefix(b *testing.B) {
	pattern := regexp.MustCompile(`^[A-Za-z0-9_]{0,15}[A-Za-z0-9]$`)
	methods := []struct {
		name     string
		validate func(string) bool
	}{
		{
			name: "byte_checks",
			validate: func(prefix string) bool {
				valid := len(prefix) >= 1 && len(prefix) <= 16
				for i := 0; valid && i < len(prefix); i++ {
					character := prefix[i]
					isLetter := character >= 'A' && character <= 'Z' || character >= 'a' && character <= 'z'
					isNumber := character >= '0' && character <= '9'
					valid = isLetter || isNumber || (character == '_' && i < len(prefix)-1)
				}
				return valid
			},
		},
		{name: "precompiled_regex", validate: pattern.MatchString},
	}
	testCases := []struct {
		name   string
		prefix string
		valid  bool
	}{
		{name: "valid", prefix: "abcdefghijklmnop", valid: true},
		{name: "invalid", prefix: "abcdefghijklmno_", valid: false},
	}

	for _, method := range methods {
		for _, testCase := range testCases {
			b.Run(method.name+"/"+testCase.name, func(b *testing.B) {
				if method.validate(testCase.prefix) != testCase.valid {
					b.Fatal("unexpected validation result")
				}

				b.ReportAllocs()
				for b.Loop() {
					benchmarkKeyV1PrefixSink = method.validate(testCase.prefix)
				}
			})
		}
	}
}

// generateKeyV1RandomRange33Benchmark maps a 33-byte number into the full Base58 output space.
func generateKeyV1RandomRange33Benchmark(reader io.Reader) ([keyV1RandomLength]byte, error) {
	random := [keyV1RandomLength]byte{}
	sampleBytes := [keyV1RandomRangeSampleBytesBenchmark]byte{}
	sample := new(big.Int)

	for {
		_, err := io.ReadFull(reader, sampleBytes[:])
		if err != nil {
			return random, err
		}

		sample.SetBytes(sampleBytes[:])
		if sample.Cmp(keyV1RandomRangeLimitBenchmark) >= 0 {
			continue
		}

		sample.Mod(sample, keyV1RandomRangeSpaceBenchmark)
		encoded := base58.Encode(sample.Bytes())
		padding := len(random) - len(encoded)
		for i := range padding {
			random[i] = '1'
		}
		copy(random[padding:], encoded)
		return random, nil
	}
}

// generateKeyV1RandomFixed32Benchmark measures Base58 encoding with leading "1" characters.
// It cannot produce every output because 32 bytes have fewer values than 44 Base58 characters.
func generateKeyV1RandomFixed32Benchmark(reader io.Reader) ([keyV1RandomLength]byte, error) {
	random := [keyV1RandomLength]byte{}
	sample := [32]byte{}
	_, err := io.ReadFull(reader, sample[:])
	if err != nil {
		return random, err
	}

	encoded := base58.Encode(sample[:])
	padding := len(random) - len(encoded)
	for i := range padding {
		random[i] = '1'
	}
	copy(random[padding:], encoded)
	return random, nil
}

// generateKeyV1RandomTruncated48Benchmark measures Base58 encoding that keeps only
// the final 44 characters. This discard makes some outputs more likely than others.
func generateKeyV1RandomTruncated48Benchmark(reader io.Reader) ([keyV1RandomLength]byte, error) {
	random := [keyV1RandomLength]byte{}
	sample := [48]byte{}
	_, err := io.ReadFull(reader, sample[:])
	if err != nil {
		return random, err
	}

	encoded := base58.Encode(sample[:])
	copy(random[:], encoded[len(encoded)-len(random):])
	return random, nil
}

// generateKeyV1RandomTruncated64Benchmark measures Base58 encoding that keeps only
// the final 44 characters. A 64-byte input always encodes to at least 64 characters.
func generateKeyV1RandomTruncated64Benchmark(reader io.Reader) ([keyV1RandomLength]byte, error) {
	random := [keyV1RandomLength]byte{}
	sample := [64]byte{}
	_, err := io.ReadFull(reader, sample[:])
	if err != nil {
		return random, err
	}

	encoded := base58.Encode(sample[:])
	copy(random[:], encoded[len(encoded)-len(random):])
	return random, nil
}
