package agent

import (
	"io"
	"time"

	"github.com/unkeyed/unkey/apps/agent/pkg/heartbeat"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
)

func setupLogging(cfg configuration) (logging.Logger, error) {

	logger := logging.New(nil)
	if cfg.Logging != nil && cfg.Logging.Axiom != nil {
		ax, err := logging.NewAxiomWriter(logging.AxiomWriterConfig{
			Token:   cfg.Logging.Axiom.Token,
			Dataset: cfg.Logging.Axiom.Dataset,
		})
		if err != nil {
			return logger, err
		}
		logger = logging.New(&logging.Config{
			Writer: []io.Writer{ax},
		})
		logger.Info().Msg("Logging to axiom")
	}
	return logger, nil
}

func setupHeartbeat(cfg configuration, logger logging.Logger) error {
	h := heartbeat.New(heartbeat.Config{
		Logger:   logger,
		Url:      cfg.Heartbeat.URL,
		Interval: time.Second * time.Duration(cfg.Heartbeat.Interval),
	})
	go h.Run()
	return nil
}
