// Package keylastusedsync implements the
// CronService.RunKeyLastUsedSync orchestrator handler. The orchestrator
// fans out to KeyLastUsedPartitionService (which is its own VO with
// persisted cursor state per partition) and waits on all partition
// futures before pinging the heartbeat.
package keylastusedsync

import (
	"fmt"
	"strconv"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	"github.com/unkeyed/unkey/pkg/logger"
)

// partitions is the number of concurrent hash-partition workers fanned
// out by the orchestrator.
const partitions = 8

// Config holds the handler's dependencies.
type Config struct {
	// Heartbeat is pinged on successful completion. Must not be nil; use
	// healthcheck.NewNoop() if monitoring is not configured.
	Heartbeat healthcheck.Heartbeat
}

// Handler executes RunKeyLastUsedSync.
type Handler struct {
	heartbeat healthcheck.Heartbeat
}

// New constructs a Handler.
func New(cfg Config) (*Handler, error) {
	if err := assert.All(
		assert.NotNil(cfg.Heartbeat, "Heartbeat must not be nil; use healthcheck.NewNoop()"),
	); err != nil {
		return nil, err
	}
	return &Handler{heartbeat: cfg.Heartbeat}, nil
}

// Handle orchestrates per-partition syncs of last_used_at from
// ClickHouse to MySQL by fanning out to KeyLastUsedPartitionService.
//
// Stateless on this side — the VO key is the fixed slug
// "key-last-used-sync" so the orchestrator runs as a singleton. Each
// partition invocation is independent and can run in parallel; this
// handler waits for all results before pinging the heartbeat.
func (h *Handler) Handle(
	ctx restate.ObjectContext,
	_ *hydrav1.RunKeyLastUsedSyncRequest,
) (*hydrav1.RunKeyLastUsedSyncResponse, error) {
	logger.Debug("running key last used sync", "partitions", partitions)

	type partitionFuture = restate.ResponseFuture[*hydrav1.SyncPartitionResponse]
	futures := make([]partitionFuture, partitions)
	for i := range partitions {
		client := hydrav1.NewKeyLastUsedPartitionServiceClient(ctx, strconv.Itoa(i))
		futures[i] = client.SyncPartition().RequestFuture(&hydrav1.SyncPartitionRequest{
			TotalPartitions: int32(partitions),
		})
	}

	var totalSynced int32
	for i, fut := range futures {
		resp, err := fut.Response()
		if err != nil {
			return nil, fmt.Errorf("partition %d error: %w", i, err)
		}
		totalSynced += resp.GetKeysSynced()
	}

	if _, err := restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
		return restate.Void{}, h.heartbeat.Ping(rc)
	}, restate.WithName("send heartbeat")); err != nil {
		return nil, fmt.Errorf("send heartbeat: %w", err)
	}

	return &hydrav1.RunKeyLastUsedSyncResponse{
		KeysSynced: totalSynced,
	}, nil
}
