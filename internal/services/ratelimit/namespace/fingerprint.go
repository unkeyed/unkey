package namespace

import (
	"crypto/sha256"
	"encoding/binary"
	"time"
)

// fingerprintVersion is hashed first, so a later encoding never collides
// with this one. Changing the encoding resets the state of every global
// limit once.
const fingerprintVersion = "ratelimit:v1"

// GlobalFingerprint identifies one global rate limit:
//
//	SHA-256("ratelimit:v1" | len‖workspaceID | len‖namespaceID | len‖identifier | duration_ms)
//
// `len‖` is a 4-byte big-endian length prefix, and `duration_ms` is 8 bytes.
// The namespace ID keeps the state when a namespace is renamed. The limit is
// left out, so an override that changes only the limit keeps what was used.
// It is safe to call concurrently.
func GlobalFingerprint(workspaceID, namespaceID, identifier string, duration time.Duration) [32]byte {
	hash := sha256.New()
	hash.Write([]byte(fingerprintVersion))
	for _, field := range []string{workspaceID, namespaceID, identifier} {
		hash.Write(binary.BigEndian.AppendUint32(nil, uint32(len(field))))
		hash.Write([]byte(field))
	}
	hash.Write(binary.BigEndian.AppendUint64(nil, uint64(duration.Milliseconds())))
	return [32]byte(hash.Sum(nil))
}
