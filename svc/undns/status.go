package undns

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
)

const (
	kindBinding = "binding"
	kindReplica = "replica"
)

var bindingStates = []reason{
	stateActive, statePendingRevision, reasonBindingUnresolved, reasonNoReadyEndpoints,
	reasonServiceMissing, reasonServiceRejected, reasonServiceRetired,
	reasonBindingInvalid, reasonBindingAmbiguous, reasonLookupError,
}

type bindingStatus struct {
	kind              string
	state             reason
	err               error
	namespace         string
	name              string
	workspace         string
	callerDeployment  string
	bindingID         string
	alias             string
	revision          string
	targetDeployment  string
	servingDeployment string
}

func newBindingStatus(config *corev1.ConfigMap, state reason, err error) bindingStatus {
	kind := kindBinding
	if deployment := config.Data["deploymentId"]; deployment != "" && deployment == config.Labels[labels.LabelKeyCallerDeploymentID] {
		kind = kindReplica
	}
	return bindingStatus{
		kind:              kind,
		state:             state,
		err:               err,
		namespace:         config.Namespace,
		name:              config.Name,
		workspace:         config.Labels[labels.LabelKeyWorkspaceID],
		callerDeployment:  config.Labels[labels.LabelKeyCallerDeploymentID],
		bindingID:         config.Labels[labels.LabelKeyBindingID],
		alias:             config.Data["appSlug"],
		revision:          config.Data["revision"],
		targetDeployment:  config.Data["deploymentId"],
		servingDeployment: config.Data["deploymentId"],
	}
}

func (s bindingStatus) serving() bool {
	return s.state == stateActive || s.state == statePendingRevision
}

type bindingChange struct {
	previous reason
	current  bindingStatus
}

func bindingChanges(previous, current map[string]bindingStatus) []bindingChange {
	var changes []bindingChange
	for key, status := range current {
		before, seen := previous[key]
		if seen && before.state == status.state {
			continue
		}
		if !seen && status.state == stateActive {
			continue
		}
		changes = append(changes, bindingChange{previous: before.state, current: status})
	}
	return changes
}

func (c bindingChange) log() {
	s := c.current
	attrs := []any{
		"namespace", s.namespace, "configmap", s.name, "kind", s.kind,
		"workspace_id", s.workspace, "caller_deployment_id", s.callerDeployment,
		"binding_id", s.bindingID, "alias", s.alias, "revision", s.revision,
		"target_deployment_id", s.targetDeployment, "serving_deployment_id", s.servingDeployment,
		"state", string(s.state), "previous_state", string(c.previous),
	}
	switch {
	case !s.serving():
		logger.Warn("private DNS binding cannot be served", append(attrs, "error", s.err)...)
	case s.state == statePendingRevision:
		logger.Info("private DNS binding serves its previous target until the new target has ready endpoints", attrs...)
	default:
		logger.Info("private DNS binding serves its target", attrs...)
	}
}

func (c *catalog) bindingCounts() map[string]map[reason]int {
	c.statusMu.Lock()
	defer c.statusMu.Unlock()
	counts := map[string]map[reason]int{kindBinding: {}, kindReplica: {}}
	for _, status := range c.statuses {
		counts[status.kind][status.state]++
	}
	return counts
}

type bindingCollector struct {
	catalog *catalog
	desc    *prometheus.Desc
}

func newBindingCollector(c *catalog) *bindingCollector {
	return &bindingCollector{
		catalog: c,
		desc: prometheus.NewDesc("unkey_dns_bindings",
			"Published binding ConfigMaps by kind and by the answer a query for them would get, from the last activation pass.",
			[]string{"kind", "state"}, nil),
	}
}

func (b *bindingCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- b.desc
}

func (b *bindingCollector) Collect(ch chan<- prometheus.Metric) {
	for kind, states := range b.catalog.bindingCounts() {
		for _, state := range bindingStates {
			ch <- prometheus.MustNewConstMetric(b.desc, prometheus.GaugeValue, float64(states[state]), kind, string(state))
		}
	}
}
