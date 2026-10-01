package undns

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
)

const (
	kindConnection = "connection"
	kindReplica    = "replica"
)

var connectionStates = []reason{
	stateActive, statePendingRevision, reasonConnectionUnresolved, reasonNoReadyEndpoints,
	reasonServiceMissing, reasonServiceRejected, reasonServiceRetired,
	reasonConnectionInvalid, reasonConnectionAmbiguous, reasonLookupError,
}

type connectionStatus struct {
	kind              string
	state             reason
	err               error
	namespace         string
	name              string
	workspace         string
	callerDeployment  string
	connectionID      string
	alias             string
	revision          string
	targetDeployment  string
	servingDeployment string
}

func newConnectionStatus(config *corev1.ConfigMap, state reason, err error) connectionStatus {
	kind := kindConnection
	if deployment := config.Data["deploymentId"]; deployment != "" && deployment == config.Labels[labels.LabelKeyCallerDeploymentID] {
		kind = kindReplica
	}
	return connectionStatus{
		kind:              kind,
		state:             state,
		err:               err,
		namespace:         config.Namespace,
		name:              config.Name,
		workspace:         config.Labels[labels.LabelKeyWorkspaceID],
		callerDeployment:  config.Labels[labels.LabelKeyCallerDeploymentID],
		connectionID:      config.Labels[labels.LabelKeyConnectionID],
		alias:             config.Data["appSlug"],
		revision:          config.Data["revision"],
		targetDeployment:  config.Data["deploymentId"],
		servingDeployment: config.Data["deploymentId"],
	}
}

func (s connectionStatus) serving() bool {
	return s.state == stateActive || s.state == statePendingRevision
}

type connectionChange struct {
	previous reason
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
		"target_deployment_id", s.targetDeployment, "serving_deployment_id", s.servingDeployment,
		"state", string(s.state), "previous_state", string(c.previous),
	}
	switch {
	case !s.serving():
		logger.Warn("private DNS connection cannot be served", append(attrs, "error", s.err)...)
	case s.state == statePendingRevision:
		logger.Info("private DNS connection serves its previous target until the new target has ready endpoints", attrs...)
	default:
		logger.Info("private DNS connection serves its target", attrs...)
	}
}

func (c *catalog) connectionCounts() map[string]map[reason]int {
	c.statusMu.Lock()
	defer c.statusMu.Unlock()
	counts := map[string]map[reason]int{kindConnection: {}, kindReplica: {}}
	for _, status := range c.statuses {
		counts[status.kind][status.state]++
	}
	return counts
}

type connectionCollector struct {
	catalog *catalog
	desc    *prometheus.Desc
}

func newConnectionCollector(c *catalog) *connectionCollector {
	return &connectionCollector{
		catalog: c,
		desc: prometheus.NewDesc("unkey_dns_connections",
			"Published connection ConfigMaps by kind and by the answer a query for them would get, from the last activation pass.",
			[]string{"kind", "state"}, nil),
	}
}

func (b *connectionCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- b.desc
}

func (b *connectionCollector) Collect(ch chan<- prometheus.Metric) {
	for kind, states := range b.catalog.connectionCounts() {
		for _, state := range connectionStates {
			ch <- prometheus.MustNewConstMetric(b.desc, prometheus.GaugeValue, float64(states[state]), kind, string(state))
		}
	}
}
