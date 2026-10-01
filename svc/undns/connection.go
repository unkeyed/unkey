package undns

import (
	"fmt"
	"maps"
	"net/netip"
	"strconv"
	"strings"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

type connection struct {
	appID      string
	deployment string
	service    string
	namespace  string
	revision   uint64
	resolved   bool
}

func (c *catalog) resolve(identity caller, app string) ([]netip.Addr, bool, error) {
	if identity.deployment == "" {
		return nil, false, nil
	}

	c.activeMu.Lock()
	defer c.activeMu.Unlock()
	key := appKey(identity.workspace, identity.project, identity.deployment, app)
	objects, err := c.connections.GetIndexer().ByIndex(appIndex, key)
	if err != nil {
		return nil, false, err
	}

	if len(objects) == 0 {
		delete(c.active, key)
		return nil, false, nil
	}
	if len(objects) != 1 {
		delete(c.active, key)
		return nil, true, errConnectionAmbiguous
	}

	config := objects[0].(*corev1.ConfigMap)
	active, err := c.activateConnection(key, config)
	if err != nil {
		return nil, true, err
	}
	addresses, err := c.resolveConnection(identity, active)
	return addresses, true, err
}

func (c *catalog) activate() {
	activations := c.activateAll()
	statuses := make(map[string]connectionStatus, len(activations))
	for _, a := range activations {
		statuses[a.config.Namespace+"/"+a.config.Name] = c.servingStatus(a.config, a.active, a.err)
	}

	c.statusMu.Lock()
	changes := connectionChanges(c.statuses, statuses)
	c.statuses = statuses
	c.statusMu.Unlock()
	for _, change := range changes {
		change.log()
	}
}

type activation struct {
	config *corev1.ConfigMap
	active *corev1.ConfigMap
	err    error
}

func (c *catalog) activateAll() []activation {
	c.activeMu.Lock()
	defer c.activeMu.Unlock()

	published := make(map[string][]*corev1.ConfigMap)
	var activations []activation
	for _, object := range c.connections.GetStore().List() {
		config := object.(*corev1.ConfigMap)
		keys, err := indexConnection(config)
		if err != nil || len(keys) != 1 {
			activations = append(activations, activation{config: config, active: nil, err: errConnectionInvalid})
			continue
		}
		published[keys[0]] = append(published[keys[0]], config)
	}

	for key, active := range c.active {
		configs := published[key]
		if len(configs) != 1 || !sameConnection(active, configs[0]) {
			delete(c.active, key)
		}
	}

	for key, configs := range published {
		if len(configs) != 1 {
			for _, config := range configs {
				activations = append(activations, activation{config: config, active: nil, err: errConnectionAmbiguous})
			}
			continue
		}
		active, err := c.activateConnection(key, configs[0])
		activations = append(activations, activation{config: configs[0], active: active, err: err})
	}
	c.activated = true
	return activations
}

func (c *catalog) servingStatus(config, active *corev1.ConfigMap, err error) connectionStatus {
	if err != nil {
		return newConnectionStatus(config, failureReason(err), err)
	}
	if _, err := c.resolveConnection(connectionCaller(active), active); err != nil {
		return newConnectionStatus(config, failureReason(err), err)
	}

	candidate, candidateErr := parseConnection(config)
	serving, servingErr := parseConnection(active)
	if candidateErr == nil && servingErr == nil && serving.revision < candidate.revision {
		status := newConnectionStatus(config, statePendingRevision, nil)
		status.servingDeployment = serving.deployment
		return status
	}
	return newConnectionStatus(config, stateActive, nil)
}

func connectionCaller(config *corev1.ConfigMap) caller {
	return caller{
		workspace:  config.Labels[labels.LabelKeyWorkspaceID],
		project:    config.Labels[labels.LabelKeyProjectID],
		kind:       "",
		deployment: config.Labels[labels.LabelKeyCallerDeploymentID],
		namespace:  config.Namespace,
	}
}

func sameConnection(a, b *corev1.ConfigMap) bool {
	return a != nil && b != nil && a.UID == b.UID && a.Namespace == b.Namespace && a.Name == b.Name &&
		a.Labels[labels.LabelKeyAppID] == b.Labels[labels.LabelKeyAppID] &&
		a.Labels[labels.LabelKeyWorkspaceID] == b.Labels[labels.LabelKeyWorkspaceID] &&
		a.Labels[labels.LabelKeyProjectID] == b.Labels[labels.LabelKeyProjectID] &&
		a.Labels[labels.LabelKeyCallerDeploymentID] == b.Labels[labels.LabelKeyCallerDeploymentID] &&
		a.Labels[labels.LabelKeyConnectionID] == b.Labels[labels.LabelKeyConnectionID] &&
		a.Data["appSlug"] == b.Data["appSlug"]
}

func (c *catalog) activateConnection(key string, config *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	candidate, err := parseConnection(config)
	if err != nil {
		delete(c.active, key)
		return nil, err
	}

	active := c.active[key]
	if !sameConnection(active, config) {
		delete(c.active, key)
		active = nil
	}
	if active != nil {
		current, err := parseConnection(active)
		if err != nil {
			return nil, err
		}
		if candidate.revision == current.revision && !maps.Equal(active.Data, config.Data) {
			return nil, fmt.Errorf("%w: data changed without advancing revision", errConnectionInvalid)
		}
		if candidate.revision <= current.revision {
			return active, nil
		}
	}

	if !candidate.resolved {
		delete(c.active, key)
		return nil, errConnectionUnresolved
	}

	if _, err := c.resolveConnection(connectionCaller(config), config); err != nil {
		if active != nil {
			return active, nil
		}
		return nil, err
	}

	if c.active == nil {
		c.active = make(map[string]*corev1.ConfigMap)
	}
	c.active[key] = config.DeepCopy()
	return c.active[key], nil
}

func parseConnection(config *corev1.ConfigMap) (connection, error) {
	revision, err := strconv.ParseUint(config.Data["revision"], 10, 64)
	if err != nil {
		return connection{}, fmt.Errorf("%w: revision: %w", errConnectionInvalid, err)
	}

	err = assert.All(
		assert.True(revision > 0, "connection revision must be positive"),
		assert.NotEmpty(config.UID),
		assert.True(config.DeletionTimestamp == nil, "connection is terminating"),
		assert.NotEmpty(config.Labels[labels.LabelKeyCallerDeploymentID], "caller deployment ID is required"),
		assert.NotEmpty(config.Labels[labels.LabelKeyConnectionID], "connection ID is required"),
		assert.NotEmpty(config.Labels[labels.LabelKeyAppID]),
		assert.Equal(config.Data["deploymentId"] == "", config.Data["serviceName"] == "", "connection target must be entirely resolved or unresolved"),
	)
	if config.Data["serviceName"] != "" {
		err = assert.All(err, assert.Equal(len(validation.IsDNS1035Label(config.Data["serviceName"])), 0, "invalid service name"))
	}
	if err != nil {
		return connection{}, fmt.Errorf("%w: %w", errConnectionInvalid, err)
	}

	return connection{
		appID:      config.Labels[labels.LabelKeyAppID],
		deployment: config.Data["deploymentId"],
		service:    config.Data["serviceName"],
		namespace:  config.Namespace,
		revision:   revision,
		resolved:   config.Data["deploymentId"] != "",
	}, nil
}

func appKey(workspace, project, callerDeployment, app string) string {
	return workspace + "/" + project + "/" + callerDeployment + "/" + app
}

func indexConnection(object any) ([]string, error) {
	config := object.(*corev1.ConfigMap)
	l := config.Labels
	app := config.Data["appSlug"]
	if l[labels.LabelKeyManagedBy] != "krane" || l[labels.LabelKeyComponent] != connectionComponent ||
		l[labels.LabelKeyWorkspaceID] == "" || l[labels.LabelKeyProjectID] == "" ||
		l[labels.LabelKeyCallerDeploymentID] == "" || l[labels.LabelKeyConnectionID] == "" ||
		l[labels.LabelKeyAppID] == "" || app == "" || strings.ToLower(app) != app || len(validation.IsDNS1123Label(app)) != 0 {
		return nil, nil
	}

	return []string{appKey(l[labels.LabelKeyWorkspaceID], l[labels.LabelKeyProjectID], l[labels.LabelKeyCallerDeploymentID], app)}, nil
}
