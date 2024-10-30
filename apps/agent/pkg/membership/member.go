package membership

import (
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
)

// utility to convert a map of tags to a Member struct
func memberFromTags(tags map[string]string) (Member, error) {
	m := Member{}
	err := m.Unmarshal(tags)
	if err != nil {
		return m, fault.Wrap(err, fmsg.With("failed to unmarshal tags"))
	}
	return m, nil
}

type Member struct {
	// Global unique identifier for the node
	NodeId   string `json:"nodeId"`
	RpcAddr  string `json:"addr"`
	SerfAddr string `json:"serfAddr"`
	State    string `json:"state"`
}

func (m *Member) Marshal() (map[string]string, error) {
	out := make(map[string]string)
	if m.NodeId == "" {
		return nil, fault.New("NodeId is empty")
	}
	out["node_id"] = m.NodeId

	if m.SerfAddr == "" {
		return nil, fault.New("SerfAddr is empty")
	}
	out["serf_addr"] = m.SerfAddr

	if m.RpcAddr == "" {
		return nil, fault.New("RpcAddr is empty")
	}
	out["rpc_addr"] = m.RpcAddr

	if m.State == "" {
		return nil, fault.New("State is empty")
	}
	out["state"] = m.State

	return out, nil
}

func (t *Member) Unmarshal(m map[string]string) error {
	var ok bool
	t.NodeId, ok = m["node_id"]
	if !ok {
		return fault.New("NodeId is missing")
	}
	t.RpcAddr, ok = m["rpc_addr"]
	if !ok {
		return fault.New("RpcAddr is missing")

	}
	t.SerfAddr, ok = m["serf_addr"]
	if !ok {
		return fault.New("SerfAddr is missing")
	}
	t.State, ok = m["state"]
	if !ok {
		return fault.New("State is missing")
	}

	return nil
}
