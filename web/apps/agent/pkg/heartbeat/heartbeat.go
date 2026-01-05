package heartbeat

import (
	"net/http"
	"time"

	"github.com/unkeyed/unkey/svc/agent/pkg/logging"
	"github.com/unkeyed/unkey/svc/agent/pkg/repeat"
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
// This function is running in a goroutine and will not block the caller.
func (h *Heartbeat) RunAsync() {
	// Tracks how many errors in a row have occurred when sending the heartbeat
	// If a heartbeat succeeds it is reset to 0
	errorsInARow := 0

	repeat.Every(h.interval, func() {
		h.logger.Debug().Msg("sending heartbeat")
		res, err := http.Post(h.url, "", nil)
		if err != nil {
			errorsInARow++
			if errorsInARow >= 3 {
				h.logger.Err(err).Int("errorsInARow", errorsInARow).Msg("error sending heartbeat")
			}
			return
		}
		errorsInARow = 0
		err = res.Body.Close()
		if err != nil {
			h.logger.Err(err).Msg("error closing response body")
		}
	})
}
