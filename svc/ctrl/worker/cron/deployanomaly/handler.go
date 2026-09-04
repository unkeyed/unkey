package deployanomaly

import (
	"fmt"
	"strconv"
	"strings"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/healthcheck"
)

const (
	windowDurationMillis = int64(5 * 60 * 1_000)
	defaultShardCount    = uint64(16)
)

// HandlerConfig holds the window orchestrator dependencies.
type HandlerConfig struct {
	Heartbeat  healthcheck.Heartbeat
	ShardCount uint64
}

// Handler fans one closed window out to stable workspace shards.
type Handler struct {
	heartbeat  healthcheck.Heartbeat
	shardCount uint64
}

// NewHandler constructs the window orchestrator.
func NewHandler(cfg HandlerConfig) (*Handler, error) {
	if err := assert.NotNil(cfg.Heartbeat, "Heartbeat must not be nil; use healthcheck.NewNoop()"); err != nil {
		return nil, err
	}
	if cfg.ShardCount == 0 {
		cfg.ShardCount = defaultShardCount
	}
	return &Handler{heartbeat: cfg.Heartbeat, shardCount: cfg.ShardCount}, nil
}

// ParseWindowStart extracts and validates the aligned unix-second window from
// a deploy anomaly cron VO key.
func ParseWindowStart(key string) (int64, error) {
	value := strings.TrimPrefix(key, "deploy-anomaly-")
	seconds, err := strconv.ParseInt(value, 10, 64)
	if err != nil || seconds <= 0 || seconds%300 != 0 {
		return 0, fault.New(fmt.Sprintf("invalid deploy anomaly window key %q", key))
	}
	return seconds * 1_000, nil
}

// Handle dispatches count-only shard calls, so the window VO never journals
// fleet metric rows.
func (h *Handler) Handle(
	ctx restate.ObjectContext,
	_ *hydrav1.RunDeployAnomalyCheckRequest,
) (*hydrav1.RunDeployAnomalyCheckResponse, error) {
	windowStart, err := ParseWindowStart(restate.Key(ctx))
	if err != nil {
		return nil, restate.TerminalError(err)
	}

	type shardFuture = restate.ResponseFuture[*hydrav1.EvaluateDeployAnomalyShardResponse]
	futures := make([]shardFuture, h.shardCount)
	for shard := range h.shardCount {
		key := ShardKey(windowStart, shard, h.shardCount)
		futures[shard] = hydrav1.NewDeployAnomalyShardServiceClient(ctx, key).
			EvaluateShard().RequestFuture(&hydrav1.EvaluateDeployAnomalyShardRequest{})
	}

	response := &hydrav1.RunDeployAnomalyCheckResponse{}
	for shard, future := range futures {
		result, responseErr := future.Response()
		if responseErr != nil {
			return nil, fault.Wrap(responseErr, fault.Internal(fmt.Sprintf("evaluate deploy anomaly shard %d", shard)))
		}
		response.GroupsDispatched += result.GetGroupsDispatched()
		response.GroupsPending += result.GetGroupsPending()
	}

	if err := restate.RunVoid(ctx, func(rc restate.RunContext) error {
		return h.heartbeat.Ping(rc)
	}, restate.WithName("send heartbeat")); err != nil {
		return nil, fault.Wrap(err, fault.Internal("send heartbeat"))
	}
	return response, nil
}

// ShardKey returns the deterministic VO key for one window partition.
func ShardKey(windowStart int64, shard, shardCount uint64) string {
	return fmt.Sprintf("%d/%d/%d", windowStart, shard, shardCount)
}

// ParseShardKey validates a shard VO key.
func ParseShardKey(key string) (windowStart int64, shard, shardCount uint64, err error) {
	parts := strings.Split(key, "/")
	if len(parts) != 3 {
		return 0, 0, 0, fault.New(fmt.Sprintf("invalid deploy anomaly shard key %q", key))
	}
	windowStart, err = strconv.ParseInt(parts[0], 10, 64)
	if err != nil || windowStart <= 0 || windowStart%windowDurationMillis != 0 {
		return 0, 0, 0, fault.New(fmt.Sprintf("invalid deploy anomaly shard window %q", key))
	}
	shard, err = strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return 0, 0, 0, fault.New(fmt.Sprintf("invalid deploy anomaly shard %q", key))
	}
	shardCount, err = strconv.ParseUint(parts[2], 10, 64)
	if err != nil || shardCount == 0 || shard >= shardCount {
		return 0, 0, 0, fault.New(fmt.Sprintf("invalid deploy anomaly shard count %q", key))
	}
	return windowStart, shard, shardCount, nil
}
