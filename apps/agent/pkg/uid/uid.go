package uid

import (
	"crypto/rand"
	"strings"

	"github.com/segmentio/ksuid"
)

type Prefix string

const (
	NodePrefix Prefix = "node"
)

// New Returns a new random base58 encoded uuid.
func New(prefix string, byteLength int) string {
	buf := make([]byte, byteLength)
	_, _ = rand.Read(buf)

	id := ksuid.New().String()
	if prefix != "" {
		return strings.Join([]string{string(prefix), id}, "_")
	} else {
		return id
	}
}
func Node() string {
	return New(string(NodePrefix), 16)
}
