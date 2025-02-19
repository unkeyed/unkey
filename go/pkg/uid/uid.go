package uid

import (
	"strings"

	"github.com/segmentio/ksuid"
)

type Prefix string

const (
	RequestPrefix           Prefix = "req"
	NodePrefix              Prefix = "node"
	RatelimitOverridePrefix Prefix = "rlor"
	TestPrefix              Prefix = "test"
)

// New Returns a new random base58 encoded uuid.
func New(prefix Prefix) string {

	id := ksuid.New().String()
	if prefix != "" {
		return strings.Join([]string{string(prefix), id}, "_")
	} else {
		return id
	}
}
func Node() string {
	return New(NodePrefix)
}

func Request() string {
	return New(RequestPrefix)
}

func Test() string {
	return New(TestPrefix)
}
