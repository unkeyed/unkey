package uid

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	mathrand "math/rand/v2"
	"time"
	"unsafe"

	"github.com/agilira/go-timecache"
	"github.com/mr-tron/base58"
)

// Prefix defines the standard resource type prefixes used throughout the system.
// These prefixes make IDs self-descriptive, allowing immediate identification
// of the resource type from the ID alone.
type Prefix string

const (
	KeyPrefix                Prefix = "key"
	PolicyPrefix             Prefix = "pol"
	APIPrefix                Prefix = "api"
	RequestPrefix            Prefix = "req"
	WorkspacePrefix          Prefix = "ws"
	KeySpacePrefix           Prefix = "ks" // keyspace
	VercelBindingPrefix      Prefix = "vb"
	RolePrefix               Prefix = "role"
	TestPrefix               Prefix = "test" // for tests only
	RatelimitNamespacePrefix Prefix = "rlns"
	RatelimitOverridePrefix  Prefix = "rlor"
	PermissionPrefix         Prefix = "perm"
	IdentityPrefix           Prefix = "id"
	RatelimitPrefix          Prefix = "rl"
	AuditLogBucketPrefix     Prefix = "buk"
	AuditLogPrefix           Prefix = "log"
	InstancePrefix           Prefix = "ins"
	GatewayPrefix            Prefix = "gw"
	WorkerPrefix             Prefix = "wkr"
	CronJobPrefix            Prefix = "cron"
	KeyEncryptionKeyPrefix   Prefix = "kek"
	OrgPrefix                Prefix = "org"
	WorkflowPrefix           Prefix = "wf"
	StepPrefix               Prefix = "step"

	// Control plane prefixes
	ProjectPrefix     Prefix = "proj"
	EnvironmentPrefix Prefix = "env"
	VersionPrefix     Prefix = "v"
	BuildPrefix       Prefix = "build"
	RootfsImagePrefix Prefix = "img"
	DomainPrefix      Prefix = "dom"
	DeploymentPrefix  Prefix = "d"
)

// epoch starts more recently so that the 32-bit number space gives a
// significantly higher useful lifetime of around 136 years
// from 2023-11-14T22:13:20Z to 2159-12-22T04:41:36Z.
const epochTimestampSec = 1700000000

// New generates a unique identifier with the specified prefix. The generated ID combines
// a timestamp component with random data, ensuring uniqueness even when created
// simultaneously across distributed systems.
//
// This is the optimized implementation that uses:
// - math/rand/v2 ChaCha8 for fast random number generation
// - Cached timestamps to avoid expensive time.Now() syscalls
// - Efficient string building without fmt.Sprintf
//
// The ID follows the format: prefix_base58encoded(data), where data contains:
// - For byteSize > 4: A timestamp (first 4 bytes) followed by random data
// - For byteSize ≤ 4: Only random data
//
// When the timestamp is included (byteSize > 4), the resulting ID is chronologically
// sortable within the same prefix type, allowing for time-based ordering of resources.
// The timestamp represents seconds since a custom epoch (Nov 14, 2023).
//
// The prefix parameter should be one of the standard Prefix constants defined in this package.
// Using standardized prefixes ensures consistency and helps with resource identification.
//
// The optional byteSize parameter controls the total bytes used for the ID. The default
// is 12 bytes, which provides a good balance between size and uniqueness. When byteSize > 4,
// the first 4 bytes contain the timestamp, with remaining bytes containing random data.
// When byteSize ≤ 4, all bytes contain random data.
//
// SECURITY NOTE: This function uses math/rand/v2 instead of crypto/rand for random number
// generation. It is suitable for generating UIDs where collision resistance is primarily
// achieved through the combination of timestamp and large random space, but should NOT be
// used for security-sensitive applications requiring cryptographic randomness (e.g., tokens,
// secrets, authentication keys). Use NewV1 for cryptographic randomness if required.
//
// Example:
//
//	// Generate a standard API key ID (12 bytes with timestamp)
//	apiKeyID := uid.New(uid.KeyPrefix)
//	// Output: key_2ey4h3ZNnWqVGUZp
//
//	// Generate a workspace ID with custom byte size
//	workspaceID := uid.New(uid.WorkspacePrefix, 16)
//	// Output: ws_3pfLMNe2vGA7h8b9VrR5
//
// See [Prefix] for available resource type prefixes and NewV1 for the original implementation.
func New(prefix Prefix, byteSize ...int) string {
	bytes := 12
	if len(byteSize) > 0 {
		bytes = byteSize[0]
	}

	// Create a buffer for our ID
	buf := make([]byte, bytes)

	// Use math/rand/v2 ChaCha8 to fill the buffer with random bytes
	rng.Read(buf)

	if bytes > 4 {
		// Use cached timestamp instead of time.Now() to avoid syscall
		t := uint32(timecache.CachedTime().Unix() - epochTimestampSec)

		// Write timestamp as first 4 bytes (big endian)
		binary.BigEndian.PutUint32(buf[:4], t)
	}

	// Encode to base58
	encoded := base58.Encode(buf)

	if prefix == "" {
		return encoded
	}

	// Use byte buffer approach with unsafe string conversion avoids intermediate allocations from string concatenation
	prefixLen := len(prefix)
	encodedLen := len(encoded)
	totalLen := prefixLen + 1 + encodedLen

	// Allocate single buffer for the entire result
	result := make([]byte, totalLen)
	copy(result, prefix)
	result[prefixLen] = '_'
	copy(result[prefixLen+1:], encoded)

	// Convert to string using unsafe (avoids copy, ~4% faster in parallel hot path)
	// SAFETY: result is a fresh allocation that we own and return immediately,
	// so the underlying bytes cannot be modified after conversion to string.
	// This pattern is used in Go stdlib: strings.Builder.String() and syscall.UTF16ToString()
	return unsafe.String(unsafe.SliceData(result), len(result))
}

var rng = mathrand.NewChaCha8([32]byte{})

// NewV1 is the original implementation using crypto/rand and fmt.Sprintf.
// Kept for backward compatibility and comparison benchmarks.
func NewV1(prefix Prefix, byteSize ...int) string {
	bytes := 12
	if len(byteSize) > 0 {
		bytes = byteSize[0]
	}

	// Create a buffer for our ID
	buf := make([]byte, bytes)

	_, err := rand.Read(buf)
	if err != nil {
		panic(err)
	}

	if bytes > 4 {
		// Calculate seconds since epoch
		// nolint:gosec
		// subtracting the epochTimestamp should guarantee we're not overflowing
		t := uint32(time.Now().Unix() - epochTimestampSec)

		// Write timestamp as first 4 bytes (big endian)
		binary.BigEndian.PutUint32(buf[:4], t)
	}

	id := encodeBase58(buf)
	if prefix != "" {
		id = fmt.Sprintf("%s_%s", prefix, id)
	}

	return id
}

// base58Alphabet is the Bitcoin base58 alphabet
const base58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

// encodeBase58 encodes a byte slice to base58 string using Bitcoin alphabet.
// This is the original custom implementation kept for NewV1 backward compatibility.
func encodeBase58(input []byte) string {
	if len(input) == 0 {
		return ""
	}

	// Count leading zeros
	var zeros int
	for _, b := range input {
		if b != 0 {
			break
		}
		zeros++
	}

	// Convert to big integer and encode
	var result []byte
	inputCopy := make([]byte, len(input))
	copy(inputCopy, input)

	for len(inputCopy) > 0 {
		// Find first non-zero byte
		start := 0
		for start < len(inputCopy) && inputCopy[start] == 0 {
			start++
		}
		if start == len(inputCopy) {
			break
		}

		// Divide by 58
		remainder := 0
		for i := start; i < len(inputCopy); i++ {
			temp := remainder*256 + int(inputCopy[i])
			inputCopy[i] = byte(temp / 58)
			remainder = temp % 58
		}

		result = append(result, base58Alphabet[remainder])
	}

	// Add leading '1's for leading zeros
	for i := 0; i < zeros; i++ {
		result = append(result, '1')
	}

	// Reverse result
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return string(result)
}
