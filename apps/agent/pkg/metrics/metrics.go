package metrics

type RingState struct {
	Nodes  int    `json:"nodes"`
	Tokens int    `json:"tokens"`
	State  string `json:"state"`
}

func (m RingState) Name() string {
	return "metric.ring.state"
}

type SystemLoad struct {
	CpuUsage float64 `json:"cpuUsage"`
	Memory   struct {
		Percentage float64 `json:"percentage"`
		Used       uint64  `json:"used"`
		Total      uint64  `json:"total"`
	} `json:"memory"`
}

func (m SystemLoad) Name() string {
	return "metric.system.load"
}
