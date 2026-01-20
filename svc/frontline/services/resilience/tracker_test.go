package resilience

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

func TestTracker_AllowsHealthyRegion(t *testing.T) {
	cfg := DefaultConfig(logging.NewNoop())
	tracker := NewTracker(cfg)

	now := time.Now()
	require.True(t, tracker.Allow("us-east-1", now))
}

func TestTracker_TripsOnHighErrorRate(t *testing.T) {
	cfg := DefaultConfig(logging.NewNoop())
	cfg.MinRequests = 5
	cfg.MaxErrorRate = 0.5
	cfg.MinErrorCount = 100
	tracker := NewTracker(cfg)

	now := time.Now()

	for i := 0; i < 5; i++ {
		tracker.Observe("us-east-1", now, Outcome{NetErr: nil, StatusCode: 502, IsInfrastructure: true})
	}
	for i := 0; i < 5; i++ {
		tracker.Observe("us-east-1", now, Outcome{NetErr: nil, StatusCode: 200, IsInfrastructure: false})
	}

	require.False(t, tracker.Allow("us-east-1", now))
}

func TestTracker_TripsOnErrorCount(t *testing.T) {
	cfg := DefaultConfig(logging.NewNoop())
	cfg.MinRequests = 100
	cfg.MinErrorCount = 3
	tracker := NewTracker(cfg)

	now := time.Now()

	for i := 0; i < 3; i++ {
		tracker.Observe("us-east-1", now, Outcome{NetErr: nil, StatusCode: 503, IsInfrastructure: true})
	}

	require.False(t, tracker.Allow("us-east-1", now))
}

func TestTracker_RecoversAfterCooldown(t *testing.T) {
	cfg := DefaultConfig(logging.NewNoop())
	cfg.MinErrorCount = 2
	cfg.Cooldown = 10 * time.Second
	tracker := NewTracker(cfg)

	now := time.Now()

	tracker.Observe("us-east-1", now, Outcome{NetErr: nil, StatusCode: 502, IsInfrastructure: true})
	tracker.Observe("us-east-1", now, Outcome{NetErr: nil, StatusCode: 502, IsInfrastructure: true})

	require.False(t, tracker.Allow("us-east-1", now))

	afterCooldown := now.Add(11 * time.Second)
	require.True(t, tracker.Allow("us-east-1", afterCooldown))
}

func TestTracker_HalfOpenProbeSuccess(t *testing.T) {
	cfg := DefaultConfig(logging.NewNoop())
	cfg.MinErrorCount = 2
	cfg.Cooldown = 10 * time.Second
	tracker := NewTracker(cfg)

	now := time.Now()

	tracker.Observe("us-east-1", now, Outcome{NetErr: nil, StatusCode: 502, IsInfrastructure: true})
	tracker.Observe("us-east-1", now, Outcome{NetErr: nil, StatusCode: 502, IsInfrastructure: true})

	afterCooldown := now.Add(11 * time.Second)
	require.True(t, tracker.Allow("us-east-1", afterCooldown))
	require.False(t, tracker.Allow("us-east-1", afterCooldown))

	tracker.Observe("us-east-1", afterCooldown, Outcome{NetErr: nil, StatusCode: 200, IsInfrastructure: false})

	require.True(t, tracker.Allow("us-east-1", afterCooldown))
}

func TestTracker_HalfOpenProbeFailure(t *testing.T) {
	cfg := DefaultConfig(logging.NewNoop())
	cfg.MinErrorCount = 2
	cfg.Cooldown = 10 * time.Second
	tracker := NewTracker(cfg)

	now := time.Now()

	tracker.Observe("us-east-1", now, Outcome{NetErr: nil, StatusCode: 502, IsInfrastructure: true})
	tracker.Observe("us-east-1", now, Outcome{NetErr: nil, StatusCode: 502, IsInfrastructure: true})

	afterCooldown := now.Add(11 * time.Second)
	require.True(t, tracker.Allow("us-east-1", afterCooldown))

	tracker.Observe("us-east-1", afterCooldown, Outcome{NetErr: nil, StatusCode: 504, IsInfrastructure: true})

	require.False(t, tracker.Allow("us-east-1", afterCooldown))
}

func TestTracker_NetworkErrorsCountAsFailures(t *testing.T) {
	cfg := DefaultConfig(logging.NewNoop())
	cfg.MinErrorCount = 2
	tracker := NewTracker(cfg)

	now := time.Now()

	tracker.Observe("us-east-1", now, Outcome{NetErr: errors.New("connection refused")})
	tracker.Observe("us-east-1", now, Outcome{NetErr: errors.New("timeout")})

	require.False(t, tracker.Allow("us-east-1", now))
}

func TestTracker_WindowResets(t *testing.T) {
	cfg := DefaultConfig(logging.NewNoop())
	cfg.Window = 5 * time.Second
	cfg.MinErrorCount = 3
	tracker := NewTracker(cfg)

	now := time.Now()

	tracker.Observe("us-east-1", now, Outcome{NetErr: nil, StatusCode: 502, IsInfrastructure: true})
	tracker.Observe("us-east-1", now, Outcome{NetErr: nil, StatusCode: 502, IsInfrastructure: true})

	afterWindow := now.Add(6 * time.Second)
	tracker.Observe("us-east-1", afterWindow, Outcome{NetErr: nil, StatusCode: 502, IsInfrastructure: true})

	require.True(t, tracker.Allow("us-east-1", afterWindow))
}

func TestTracker_IsolatesRegions(t *testing.T) {
	cfg := DefaultConfig(logging.NewNoop())
	cfg.MinErrorCount = 2
	tracker := NewTracker(cfg)

	now := time.Now()

	tracker.Observe("us-east-1", now, Outcome{NetErr: nil, StatusCode: 502, IsInfrastructure: true})
	tracker.Observe("us-east-1", now, Outcome{NetErr: nil, StatusCode: 502, IsInfrastructure: true})

	require.False(t, tracker.Allow("us-east-1", now))
	require.True(t, tracker.Allow("us-west-2", now))
}

func TestTracker_IgnoresUserApp5xx(t *testing.T) {
	cfg := DefaultConfig(logging.NewNoop())
	cfg.MinErrorCount = 2
	tracker := NewTracker(cfg)

	now := time.Now()

	tracker.Observe("us-east-1", now, Outcome{NetErr: nil, StatusCode: 500, IsInfrastructure: false})
	tracker.Observe("us-east-1", now, Outcome{NetErr: nil, StatusCode: 500, IsInfrastructure: false})
	tracker.Observe("us-east-1", now, Outcome{NetErr: nil, StatusCode: 500, IsInfrastructure: false})

	require.True(t, tracker.Allow("us-east-1", now))
}
