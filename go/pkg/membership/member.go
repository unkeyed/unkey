package membership

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// Member represents a node in the cluster with its identifying information
// and network address.
type Member struct {
	// NodeID is a globally unique identifier for the node
	NodeID string `json:"nodeId"`
	// IP Address or DNS name where this node can be reached
	Host string `json:"host"`

	GossipPort int `json:"gossipPort"`
	RpcPort    int `json:"rpcPort"`
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

// ToMap converts a Member into a map[string]any representation
func (m Member) ToMap() map[string]string {
	return map[string]string{
		"nodeId":     m.NodeID,
		"host":       m.Host,
		"gossipPort": fmt.Sprintf("%d", m.GossipPort),
		"rpcPort":    fmt.Sprintf("%d", m.RpcPort),
	}

}

func memberFromMap(m map[string]string) (Member, error) {
	// nolint:exhaustruct
	member := Member{}

	if nodeID, ok := m["nodeId"]; ok {
		member.NodeID = nodeID
	} else {
		return Member{}, fmt.Errorf("missing nodeId field")
	}

	if host, ok := m["host"]; ok {
		member.Host = host
	} else {
		return Member{}, fmt.Errorf("missing host field")
	}

	if gossipPortStr, ok := m["gossipPort"]; ok {
		var err error
		member.GossipPort, err = strconv.Atoi(gossipPortStr)
		if err != nil {
			return Member{}, fmt.Errorf("invalid gossipPort: %w", err)
		}
	}

	if rpcPortStr, ok := m["rpcPort"]; ok {
		var err error
		member.RpcPort, err = strconv.Atoi(rpcPortStr)
		if err != nil {
			return Member{}, fmt.Errorf("invalid rpcPort: %w", err)
		}
	}

	return member, nil
}
