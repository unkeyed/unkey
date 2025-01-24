package uid

import (
	"strings"

	"github.com/segmentio/ksuid"
)

type Prefix string

const (
	RequestPrefix Prefix = "req"
	NodePrefix    Prefix = "node"
)

// New Returns a new random base58 encoded uuid.
func New(prefix string) string {

	id := ksuid.New().String()
	if prefix != "" {
		return strings.Join([]string{prefix, id}, "_")
	} else {
		return id
	}
}
func Node() string {
	return New(string(NodePrefix))
}

func Request() string {
	return New(string(RequestPrefix))
}
