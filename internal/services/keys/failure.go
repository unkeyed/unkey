package keys

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"math"
	"strings"
	"sync"

	"github.com/unkeyed/unkey/internal/services/keys/metrics"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/hash"
)

// failureSampleRate is the weight of a row kept above the cap: one row in
// failureSampleRate is written and stands for that many failures.
const failureSampleRate = 100

// FailureRecorderConfig configures a [FailureRecorder].
type FailureRecorderConfig struct {
	// Buffer receives the rows. It should drop rows when full so a slow
	// ClickHouse never blocks verifications.
	Buffer *batch.BatchProcessor[schema.KeyVerificationFailure]

	// FingerprintSecret keys the HMAC behind KeyFingerprint. Every pod must use
	// the same secret so fingerprints match across pods and regions. When it is
	// empty, KeyFingerprint is 0.
	FingerprintSecret string

	// RowsPerSecond is the number of full-weight rows one pod writes per
	// second. Above it, one row in 100 is written with a sample weight of 100.
	RowsPerSecond int

	Clock clock.Clock
}

// FailureRecorder writes key verification failures to the internal
// key_verification_failures stream. It caps the rows per pod and samples
// above the cap, so a flood of bad keys cannot flood ClickHouse.
type FailureRecorder struct {
	buffer        *batch.BatchProcessor[schema.KeyVerificationFailure]
	secret        []byte
	rowsPerSecond int
	clock         clock.Clock

	mu           sync.Mutex
	windowStart  int64
	windowRows   int
	overflowRows int
}

// NewFailureRecorder creates a [FailureRecorder].
func NewFailureRecorder(cfg FailureRecorderConfig) *FailureRecorder {
	return &FailureRecorder{
		buffer:        cfg.Buffer,
		secret:        []byte(cfg.FingerprintSecret),
		rowsPerSecond: cfg.RowsPerSecond,
		clock:         cfg.Clock,
		mu:            sync.Mutex{},
		windowStart:   0,
		windowRows:    0,
		overflowRows:  0,
	}
}

// Record buffers row with the shape of rawKey, the key the caller submitted.
// The raw key and its hash never leave this function; the row only gets the
// length, a capped prefix, the character set, and a keyed fingerprint.
func (r *FailureRecorder) Record(row schema.KeyVerificationFailure, rawKey string) {
	weight, result := r.admit()
	metrics.KeyVerificationFailureRowsTotal.WithLabelValues(result).Inc()
	if weight == 0 {
		return
	}

	shape := DescribeKey(rawKey)
	row.KeyLength = shape.Length
	row.KeyPrefix = shape.Prefix
	row.KeyCharset = shape.Charset
	row.KeyFingerprint = r.fingerprint(row.WorkspaceID, hash.Sha256(rawKey))
	row.SampleWeight = weight
	r.buffer.Buffer(row)
}

// admit applies the per-second cap and returns the row's sample weight, 0 when
// the row must not be written, and the result label for the rows counter.
func (r *FailureRecorder) admit() (uint32, string) {
	second := r.clock.Now().Unix()

	r.mu.Lock()
	defer r.mu.Unlock()

	if second != r.windowStart {
		r.windowStart = second
		r.windowRows = 0
	}
	if r.windowRows < r.rowsPerSecond {
		r.windowRows++
		return 1, "recorded"
	}

	r.overflowRows++
	if r.overflowRows%failureSampleRate == 0 {
		return failureSampleRate, "sampled"
	}
	return 0, "sampled_out"
}

// fingerprint returns the first 8 bytes of HMAC-SHA256 over the workspace and
// the stored key hash. The workspace scope gives one key a different
// fingerprint in every workspace, so rows cannot link a key across tenants.
// The secret prevents testing guessed keys against fingerprints. Someone with
// the secret can still fingerprint a hash from MySQL to find its failures.
func (r *FailureRecorder) fingerprint(workspaceID, keyHash string) uint64 {
	if len(r.secret) == 0 {
		return 0
	}
	mac := hmac.New(sha256.New, r.secret)
	mac.Write([]byte(workspaceID))
	mac.Write([]byte{0})
	mac.Write([]byte(keyHash))
	return binary.BigEndian.Uint64(mac.Sum(nil))
}

// KeyShape describes a submitted key without revealing it.
type KeyShape struct {
	// Length is the key length in bytes, capped at 65535.
	Length uint16

	// Prefix is the text before the first underscore when it is 1 to 8 ASCII
	// letters or digits and the key is at least 16 bytes long. Otherwise it is
	// empty, so short keys and keys without a prefix reveal nothing here.
	Prefix string

	// Charset classifies the key after the prefix: "hex", "base58", "alnum",
	// "base64url", "other", or "empty".
	Charset string
}

// DescribeKey returns the shape of rawKey.
func DescribeKey(rawKey string) KeyShape {
	shape := KeyShape{
		Length:  uint16(min(len(rawKey), math.MaxUint16)),
		Prefix:  "",
		Charset: "",
	}

	body := rawKey
	if i := strings.IndexByte(rawKey, '_'); len(rawKey) >= 16 && i >= 1 && i <= 8 && onlyChars(rawKey[:i], alnumChars) {
		shape.Prefix = rawKey[:i]
		body = rawKey[i+1:]
	}

	switch {
	case body == "":
		shape.Charset = "empty"
	case onlyChars(body, hexChars):
		shape.Charset = "hex"
	case onlyChars(body, base58Chars):
		shape.Charset = "base58"
	case onlyChars(body, alnumChars):
		shape.Charset = "alnum"
	case onlyChars(body, alnumChars+"-_"):
		shape.Charset = "base64url"
	default:
		shape.Charset = "other"
	}
	return shape
}

const (
	hexChars    = "0123456789abcdefABCDEF"
	base58Chars = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	alnumChars  = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

func onlyChars(s, chars string) bool {
	for i := 0; i < len(s); i++ {
		if strings.IndexByte(chars, s[i]) < 0 {
			return false
		}
	}
	return true
}
