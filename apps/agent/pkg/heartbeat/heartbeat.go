package heartbeat

import (
	"net/http"
	"time"

	"github.com/unkeyed/unkey/apps/vault/pkg/logging"
)

type Heartbeat struct {
	logger   logging.Logger
	url      string
	interval time.Duration
}

type Config struct {
	Logger   logging.Logger
	Url      string
	Interval time.Duration
}

func New(config Config) *Heartbeat {

	h := &Heartbeat{
		url:      config.Url,
		logger:   config.Logger,
		interval: config.Interval,
	}
	if h.interval == 0 {
		h.interval = time.Minute
	}
	return h

}

// Starts a timer that sends a POST request to the URL every interval
// This function is blocking, run it in a go routine yourself:
// example: go h.Run()
func (h *Heartbeat) Run() {
	t := time.NewTicker(h.interval)
	defer t.Stop()

	for range t.C {
		h.logger.Info().Msg("sending heartbeat")
		res, err := http.Post(h.url, "", nil)
		if err != nil {
			h.logger.Err(err).Msg("error sending heartbeat")
			continue
		}
		err = res.Body.Close()
		if err != nil {
			h.logger.Err(err).Msg("error closing response body")
		}
	}

}
