package membership

import (
	"encoding/json"
)

// Member represents a node in the cluster with its identifying information
// and network address.
type Member struct {
	// NodeID is a globally unique identifier for the node
	NodeID string `json:"nodeId"`
	// Addr is the network address of the node
	Addr string `json:"addr"`
}

// Marshal encodes the Member into a JSON byte slice.
// Returns the encoded bytes or an error if marshaling fails.
func (m Member) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

// Unmarshal decodes a JSON byte slice into the Member.
// Returns an error if unmarshaling fails.
func (m *Member) Unmarshal(b []byte) error {
	return json.Unmarshal(b, m)
}
