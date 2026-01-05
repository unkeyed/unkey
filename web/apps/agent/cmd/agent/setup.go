package agent

import (
	"io"
	"time"

	"github.com/unkeyed/unkey/svc/agent/pkg/config"
	"github.com/unkeyed/unkey/svc/agent/pkg/heartbeat"
	"github.com/unkeyed/unkey/svc/agent/pkg/logging"
	"github.com/unkeyed/unkey/svc/agent/pkg/uid"
)

func setupLogging(cfg config.Agent) (logging.Logger, error) {

	logger := logging.New(nil)

	// runId is unique per start of the agent, this is useful for differnetiating logs between
	// deployments
	// If the agent is restarted, the runId will change
	logger = logger.With().Str("runId", uid.New("run")).Logger()

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

func setupHeartbeat(cfg config.Agent, logger logging.Logger) {
	h := heartbeat.New(heartbeat.Config{
		Logger:   logger,
		Url:      cfg.Heartbeat.URL,
		Interval: time.Second * time.Duration(cfg.Heartbeat.Interval),
	})
	h.RunAsync()

}
