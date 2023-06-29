package uid

import (
	"crypto/rand"
	"github.com/btcsuite/btcd/btcutil/base58"
	"strings"
)

const encodingVersion = 0x01

type Prefix string

const (
	WorkspacePrefix Prefix = "ws"
	KeyPrefix       Prefix = "key"
	ApiPrefix       Prefix = "api"
	UnkeyPrefix     Prefix = "unkey"
)

// New Returns a new random base58 encoded uuid.
func New(byteLength int, prefix string) string {
	buf := make([]byte, byteLength)
	rand.Read(buf)
	r := base58.CheckEncode(buf, encodingVersion)

	if prefix != "" {
		return strings.Join([]string{string(prefix), r}, "_")
	} else {
		return r
	}

}

func Workspace() string {
	return New(16, string(WorkspacePrefix))
}

func Key() string {
	return New(16, string(KeyPrefix))
}

func Api() string {
	return New(16, string(ApiPrefix))
}
