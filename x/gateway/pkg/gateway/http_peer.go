package gateway

import "time"

type Upstream struct {
	addr    string
	timeout time.Duration
}
