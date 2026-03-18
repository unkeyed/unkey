package keylastusedsync

import (
	"fmt"
	"strconv"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/logger"
)

// defaultPartitions is the number of concurrent hash-partition workers.
const defaultPartitions = 8

// RunSync is the orchestrator that fans out SyncPartition calls to each partition
// worker and collects results. Each partition is a separate Restate virtual object
// with its own key, journal, and persisted cursor state.
func (s *Service) RunSync(
	ctx restate.ObjectContext,
	_ *hydrav1.RunSyncRequest,
) (*hydrav1.RunSyncResponse, error) {
	runKey := restate.Key(ctx)
	logger.Info("running key last used sync", "run", runKey, "partitions", defaultPartitions)

	// Fan out to partition services — each is an independent virtual object
	type partitionFuture = restate.ResponseFuture[*hydrav1.SyncPartitionResponse]
	futures := make([]partitionFuture, defaultPartitions)
	for i := range defaultPartitions {
		client := hydrav1.NewKeyLastUsedPartitionServiceClient(ctx, strconv.Itoa(i))
		futures[i] = client.SyncPartition().RequestFuture(&hydrav1.SyncPartitionRequest{
			TotalPartitions: int32(defaultPartitions), //nolint:gosec
		})
	}

	// Collect results
	var totalSynced int32
	for i, fut := range futures {
		resp, err := fut.Response()
		if err != nil {
			return nil, fmt.Errorf("partition %d error: %w", i, err)
		}
		totalSynced += resp.GetKeysSynced()
	}

	logger.Info("key last used sync complete",
		"run", runKey,
		"keys_synced", totalSynced,
	)

	_, err := restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
		return restate.Void{}, s.heartbeat.Ping(rc)
	}, restate.WithName("send heartbeat"))
	if err != nil {
		return nil, fmt.Errorf("send heartbeat: %w", err)
	}

	return &hydrav1.RunSyncResponse{
		KeysSynced: totalSynced,
	}, nil
}
