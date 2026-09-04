package deployanomaly_test

import (
	"net/url"
	"sync"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	restatetest "github.com/restatedev/sdk-go/testing"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/deployanomaly"
)

type handoffTestShard struct {
	hydrav1.UnimplementedDeployAnomalyShardServiceServer

	firstKey      string
	started       chan struct{}
	secondStarted chan struct{}
	release       chan struct{}
	firstOnce     sync.Once
	secondOnce    sync.Once
}

func (h *handoffTestShard) EvaluateShard(
	ctx restate.ObjectContext,
	_ *hydrav1.EvaluateDeployAnomalyShardRequest,
) (*hydrav1.EvaluateDeployAnomalyShardResponse, error) {
	if restate.Key(ctx) == h.firstKey {
		h.firstOnce.Do(func() { close(h.started) })
		<-h.release
		restate.Set(ctx, "pending", true)
		return &hydrav1.EvaluateDeployAnomalyShardResponse{GroupsPending: 1}, nil
	}
	h.secondOnce.Do(func() { close(h.secondStarted) })

	previous, err := hydrav1.NewDeployAnomalyShardServiceClient(ctx, h.firstKey).
		GetPending().Request(&hydrav1.GetPendingDeployAnomalyGroupsRequest{})
	if err != nil {
		return nil, err
	}
	return &hydrav1.EvaluateDeployAnomalyShardResponse{GroupsPending: int32(len(previous.GetGroups()))}, nil
}

func (h *handoffTestShard) GetPending(
	ctx restate.ObjectContext,
	_ *hydrav1.GetPendingDeployAnomalyGroupsRequest,
) (*hydrav1.GetPendingDeployAnomalyGroupsResponse, error) {
	pending, err := restate.Get[bool](ctx, "pending")
	if err != nil {
		return nil, err
	}
	response := &hydrav1.GetPendingDeployAnomalyGroupsResponse{}
	if pending {
		response.Groups = []*hydrav1.DeployAnomalyGroupKey{{WorkspaceId: "ws"}}
	}
	return response, nil
}

func TestReverseDeliveryWaitsForPreviousAnomalyShard(t *testing.T) {
	windowStart := time.Now().UTC().Truncate(5 * time.Minute).UnixMilli()
	firstKey := deployanomaly.ShardKey(windowStart, 0, 1)
	secondKey := deployanomaly.ShardKey(windowStart+5*time.Minute.Milliseconds(), 0, 1)
	server := &handoffTestShard{
		firstKey:      firstKey,
		started:       make(chan struct{}),
		secondStarted: make(chan struct{}),
		release:       make(chan struct{}),
	}
	testEnv := restatetest.Start(t, hydrav1.NewDeployAnomalyShardServiceServer(server))

	type result struct {
		response *hydrav1.EvaluateDeployAnomalyShardResponse
		err      error
	}
	firstResult := make(chan result, 1)
	go func() {
		response, err := hydrav1.NewDeployAnomalyShardServiceIngressClient(testEnv.Ingress(), url.PathEscape(firstKey)).
			EvaluateShard().Request(t.Context(), &hydrav1.EvaluateDeployAnomalyShardRequest{})
		firstResult <- result{response: response, err: err}
	}()
	select {
	case <-server.started:
	case <-time.After(10 * time.Second):
		t.Fatal("first anomaly shard did not start")
	}

	secondResult := make(chan result, 1)
	go func() {
		response, err := hydrav1.NewDeployAnomalyShardServiceIngressClient(testEnv.Ingress(), url.PathEscape(secondKey)).
			EvaluateShard().Request(t.Context(), &hydrav1.EvaluateDeployAnomalyShardRequest{})
		secondResult <- result{response: response, err: err}
	}()
	select {
	case <-server.secondStarted:
	case <-time.After(10 * time.Second):
		close(server.release)
		<-firstResult
		t.Fatal("later anomaly shard did not start")
	}
	select {
	case result := <-secondResult:
		close(server.release)
		<-firstResult
		require.NoError(t, result.err)
		t.Fatal("later shard overtook the in-flight previous window")
	case <-time.After(250 * time.Millisecond):
	}

	close(server.release)
	first := <-firstResult
	require.NoError(t, first.err)
	require.Equal(t, int32(1), first.response.GetGroupsPending())
	second := <-secondResult
	require.NoError(t, second.err)
	require.Equal(t, int32(1), second.response.GetGroupsPending())
}
