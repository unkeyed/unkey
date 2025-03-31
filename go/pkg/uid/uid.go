package uid

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/btcsuite/btcutil/base58"
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
	KeyAuthPrefix            Prefix = "ks" // keyspace
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
)

// epoch starts more recently so that the 32-bit number space gives a
// significantly higher useful lifetime of around 136 years
// from 2023-11-14T22:13:20Z to 2159-12-22T04:41:36Z.

const epochTimestampSec = 1700000000

// New generates a unique identifier with an optional prefix.
// It creates a KSUID (K-Sortable Unique Identifier) and prepends the
// specified prefix with an underscore separator if a prefix is provided.
//
// KSUIDs are 27-character, URL-safe, base62-encoded strings that contain:
// - A timestamp with 1-second resolution (first 4 bytes)
// - 16 bytes of random data
//
// This makes them:
// - Time sortable (newer KSUIDs sort lexicographically after older ones)
// - Highly unique (with ~2^128 possible random combinations)
// - More compact than UUIDs
//
// Example:
//
//	// Generate an ID with a custom prefix
//	id := uid.New("invoice") // returns "invoice_1z4UVH4CbRPvgSfCBmheK2h8xZb"
//
//	// Generate an ID without a prefix
//	id := uid.New("") // returns "1z4UVH4CbRPvgSfCBmheK2h8xZb"
func New(prefix Prefix) string {

	// Create a buffer for our ID
	buf := make([]byte, 12)

	// Calculate seconds since epoch
	// nolint:gosec
	// subtracting the epochTimestamp should guarantee we're not overflowing
	t := uint32(time.Now().Unix() - epochTimestampSec)

	// Write timestamp as first 4 bytes (big endian)
	binary.BigEndian.PutUint32(buf[:4], t)

	// Fill remaining 8 bytes with random data
	_, err := rand.Read(buf[4:])
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf("%s_%s", prefix, base58.Encode(buf))
}
