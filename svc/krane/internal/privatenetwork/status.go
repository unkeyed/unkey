package privatenetwork

import (
	"time"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/krane/pkg/metrics"
)

const (
	loopDiscovery = "discovery"
	loopEndpoints = "endpoints"

	stageList          = "list"
	stageSnapshot      = "snapshot"
	stageInvalidEntry  = "invalid_entry"
	stageNamespace     = "namespace"
	stageService       = "service"
	stageEndpointSlice = "endpoint_slice"
	stagePolicy        = "policy"
	stageBinding       = "binding"
	stageCleanup       = "cleanup"

	kindBinding = "binding"
	kindReplica = "replica"

	stateCurrent             = "current"
	stateWaitingForEndpoints = "waiting_for_endpoints"
	stateUnresolved          = "unresolved"
	stateFailed              = "failed"
)

var (
	loopStages = map[string][]string{
		loopDiscovery: {stageList, stageSnapshot, stageInvalidEntry, stageNamespace, stageService, stageEndpointSlice, stagePolicy, stageBinding, stageCleanup},
		loopEndpoints: {stageList, stageEndpointSlice},
	}
	entryKinds  = []string{kindBinding, kindReplica}
	entryStates = []string{stateCurrent, stateWaitingForEndpoints, stateUnresolved, stateFailed}
)

func initMetrics() {
	metrics.PrivateNetworkLeader.Set(0)
	for loop, stages := range loopStages {
		for _, stage := range stages {
			metrics.PrivateNetworkErrorsTotal.WithLabelValues(loop, stage)
		}
	}
	for _, kind := range entryKinds {
		for _, state := range entryStates {
			metrics.PrivateNetworkEntries.WithLabelValues(kind, state).Set(0)
		}
	}
}

func startLeading() {
	metrics.PrivateNetworkLeader.Set(1)
	for loop := range loopStages {
		metrics.PrivateNetworkLastCompletedPassUnixSeconds.WithLabelValues(loop).Set(0)
	}
}

func countError(loop, stage string, err error) error {
	metrics.PrivateNetworkErrorsTotal.WithLabelValues(loop, stage).Inc()
	return err
}

func observePass(loop string, started time.Time, completed bool) {
	metrics.PrivateNetworkPassDurationSeconds.WithLabelValues(loop).Observe(time.Since(started).Seconds())
	if completed {
		metrics.PrivateNetworkLastCompletedPassUnixSeconds.WithLabelValues(loop).Set(float64(time.Now().Unix()))
	}
}

type entryStatus struct {
	kind                string
	state               string
	stage               string
	err                 error
	app                 *ctrlv1.PrivateNetworkApp
	publishedDeployment string
}

func entryKind(app *ctrlv1.PrivateNetworkApp) string {
	if app != nil && app.GetDeploymentId() != "" && app.GetDeploymentId() == app.GetCallerDeploymentId() {
		return kindReplica
	}
	return kindBinding
}

func recordEntries(entries map[string]entryStatus, untracked []entryStatus) {
	counts := make(map[string]map[string]int, len(entryKinds))
	for _, kind := range entryKinds {
		counts[kind] = make(map[string]int, len(entryStates))
	}
	for _, entry := range entries {
		counts[entry.kind][entry.state]++
	}
	for _, entry := range untracked {
		counts[entry.kind][entry.state]++
	}
	for _, kind := range entryKinds {
		for _, state := range entryStates {
			metrics.PrivateNetworkEntries.WithLabelValues(kind, state).Set(float64(counts[kind][state]))
		}
	}
}

func logEntryChanges(previous, current map[string]entryStatus) {
	for key, entry := range current {
		before := previous[key]
		failing := entry.state == stateFailed && (before.state != stateFailed || before.stage != entry.stage)
		recovered := before.state == stateFailed && entry.state != stateFailed
		waiting := entry.state == stateWaitingForEndpoints && before.state != stateWaitingForEndpoints
		if !failing && !recovered && !waiting {
			continue
		}

		attrs := []any{
			"binding_key", key, "kind", entry.kind, "state", entry.state, "previous_state", before.state,
			"workspace_id", entry.app.GetWorkspaceId(), "binding_id", entry.app.GetBindingId(),
			"caller_deployment_id", entry.app.GetCallerDeploymentId(), "alias", entry.app.GetBindingName(),
			"target_deployment_id", entry.app.GetDeploymentId(), "published_deployment_id", entry.publishedDeployment,
		}
		switch entry.state {
		case stateFailed:
			logger.Warn("private network entry failed to publish; its previous objects stay published",
				append(attrs, "stage", entry.stage, "error", entry.err)...)
		case stateWaitingForEndpoints:
			logger.Info("private network binding keeps its previous target until the new target has ready endpoints", attrs...)
		default:
			logger.Info("private network entry recovered", attrs...)
		}
	}
}
