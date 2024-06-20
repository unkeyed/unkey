package membership

import "fmt"

type Tags struct {
	SerfAddr        string
	NodeId          string
	Region string
}

func (t *Tags) Marshal() (map[string]string, error) {
	m := make(map[string]string)
	if t.SerfAddr == "" {
		return nil, fmt.Errorf("SerfAddr is empty")
	}
	m["serf_addr"] = t.SerfAddr

	if t.NodeId == "" {
		return nil, fmt.Errorf("NodeId is empty")
	}
	m["node_id"] = t.NodeId

	

	if t.Region == "" {
		return nil, fmt.Errorf("Region is empty")
	}
	m["region"] = t.Region

	return m, nil
}

func (t *Tags) Unmarshal(m map[string]string) error {
	var ok bool
	t.SerfAddr, ok = m["serf_addr"]
	if !ok {
		return fmt.Errorf("serf_addr is empty")
	}
	t.NodeId, ok = m["node_id"]
	if !ok {
		return fmt.Errorf("node_id is empty")
	}

	t.Region, ok = m["region"]
	if !ok {
		return fmt.Errorf("region is empty")
	}
	return nil
}
