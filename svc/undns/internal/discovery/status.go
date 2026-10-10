package discovery

import (
	"github.com/unkeyed/unkey/pkg/deploy/privatenetwork"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/undns/pkg/metrics"
	corev1 "k8s.io/api/core/v1"
)

const (
	kindConnection = "connection"
	kindReplica    = "replica"

	stateActive Reason = "active"
)

var connectionStates = []Reason{
	stateActive, ReasonConnectionUnresolved, ReasonNoReadyEndpoints,
	ReasonServiceMissing, ReasonServiceRejected, ReasonServiceRetired,
	ReasonConnectionInvalid, ReasonConnectionAmbiguous, ReasonLookupError,
}

type connectionStatus struct {
	kind             string
	state            Reason
	err              error
	namespace        string
	name             string
	workspace        string
	callerDeployment string
	connectionID     string
	alias            string
	revision         string
	targetDeployment string
}

func newConnectionStatus(config *corev1.ConfigMap, state Reason, err error) connectionStatus {
	kind := kindConnection
	data, decodeErr := privatenetwork.Decode(config.Data)
	if decodeErr != nil {
		state, err = ReasonConnectionInvalid, fail(ReasonConnectionInvalid, "invalid app connection: %w", decodeErr)
	}
	if data.DeploymentID != "" && data.DeploymentID == config.Labels[privatenetwork.CallerDeploymentLabel] {
		kind = kindReplica
	}

	return connectionStatus{
		kind:             kind,
		state:            state,
		err:              err,
		namespace:        config.Namespace,
		name:             config.Name,
		workspace:        config.Labels[privatenetwork.WorkspaceLabel],
		callerDeployment: config.Labels[privatenetwork.CallerDeploymentLabel],
		connectionID:     config.Labels[privatenetwork.ConnectionLabel],
		alias:            config.Data[privatenetwork.ConnectionAliasKey],
		revision:         config.Data[privatenetwork.ConnectionRevisionKey],
		targetDeployment: data.DeploymentID,
	}
}

func (s connectionStatus) serving() bool {
	return s.state == stateActive
}

type connectionChange struct {
	previous Reason
	current  connectionStatus
}

func connectionChanges(previous, current map[string]connectionStatus) []connectionChange {
	var changes []connectionChange
	for key, status := range current {
		before, seen := previous[key]
		if seen && before.state == status.state {
			continue
		}
		if !seen && status.state == stateActive {
			continue
		}
		changes = append(changes, connectionChange{previous: before.state, current: status})
	}
	return changes
}

func (c connectionChange) log() {
	s := c.current
	attrs := []any{
		"namespace", s.namespace, "configmap", s.name, "kind", s.kind,
		"workspace_id", s.workspace, "caller_deployment_id", s.callerDeployment,
		"connection_id", s.connectionID, "alias", s.alias, "revision", s.revision,
		"target_deployment_id", s.targetDeployment,
		"state", string(s.state), "previous_state", string(c.previous),
	}
	switch {
	case !s.serving():
		logger.Warn("private DNS connection cannot be served", append(attrs, "error", s.err)...)
	default:
		logger.Info("private DNS connection serves its target", attrs...)
	}
}

func reportConnections(counts map[string]map[Reason]int) {
	for _, kind := range []string{kindConnection, kindReplica} {
		for _, state := range connectionStates {
			metrics.Connections.WithLabelValues(kind, string(state)).Set(float64(counts[kind][state]))
		}
	}
}
