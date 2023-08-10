package uid

import (
	"crypto/rand"
	"github.com/btcsuite/btcd/btcutil/base58"
	"strings"
)

type Prefix string

const (
	WorkspacePrefix Prefix = "ws"
	KeyPrefix       Prefix = "key"
	ApiPrefix       Prefix = "api"
	UnkeyPrefix     Prefix = "unkey"
	KeyAuthPrefix   Prefix = "key_auth"
)

// New Returns a new random base58 encoded uuid.
func New(byteLength int, prefix string) string {
	buf := make([]byte, byteLength)
	_, _ = rand.Read(buf)
	r := base58.Encode(buf)

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

func KeyAuth() string {
	return New(16, string(KeyAuthPrefix))
}
