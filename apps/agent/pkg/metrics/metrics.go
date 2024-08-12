package metrics

type RingState struct {
	Nodes  int    `json:"nodes"`
	Tokens int    `json:"tokens"`
	State  string `json:"state"`
}

func (m RingState) Name() string {
	return "metric.ring.state"
}
