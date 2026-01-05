package profiling

import (
	"runtime"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	"github.com/grafana/pyroscope-go"
	"github.com/unkeyed/unkey/svc/agent/pkg/config"
	"github.com/unkeyed/unkey/svc/agent/pkg/logging"
)

func Start(cfg config.Agent, logger logging.Logger) error {
	if cfg.Pyroscope == nil {
		logger.Info().Msg("profiling is disabled")
		return nil
	}
	runtime.SetMutexProfileFraction(5)
	runtime.SetBlockProfileRate(5)

	_, err := pyroscope.Start(pyroscope.Config{
		UploadRate:        time.Minute,
		ApplicationName:   "api.unkey.cloud",
		ServerAddress:     cfg.Pyroscope.Url,
		BasicAuthUser:     cfg.Pyroscope.User,
		BasicAuthPassword: cfg.Pyroscope.Password,
		Tags: map[string]string{
			"nodeId": cfg.NodeId,
			"image":  cfg.Image,
			"region": cfg.Region,
		},
		// Logger: pyroscope.StandardLogger,

		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileInuseObjects,
			pyroscope.ProfileInuseSpace,

			pyroscope.ProfileGoroutines,
			pyroscope.ProfileMutexCount,
			pyroscope.ProfileMutexDuration,
			pyroscope.ProfileBlockCount,
			pyroscope.ProfileBlockDuration,
		},
	})

	if err != nil {
		return fault.Wrap(err, fmsg.With("unable to start profiling"))
	}

	logger.Info().Msg("sending profiles to grafana")
	return nil
}
