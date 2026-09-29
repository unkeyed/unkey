package undns

import (
	"fmt"
	"maps"
	"net/netip"
	"strconv"
	"strings"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

type binding struct {
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
	objects, err := c.bindings.GetIndexer().ByIndex(appIndex, key)
	if err != nil {
		return nil, false, err
	}

	if len(objects) == 0 {
		delete(c.active, key)
		return nil, false, nil
	}
	if len(objects) != 1 {
		delete(c.active, key)
		return nil, true, fmt.Errorf("ambiguous app binding")
	}

	config := objects[0].(*corev1.ConfigMap)
	active, err := c.activateBinding(key, config)
	if err != nil {
		return nil, true, err
	}
	addresses, err := c.resolveBinding(identity, active)
	return addresses, true, err
}

func (c *catalog) activate() {
	c.activeMu.Lock()
	defer c.activeMu.Unlock()

	desired := make(map[string]*corev1.ConfigMap)
	ambiguous := make(map[string]bool)
	for _, object := range c.bindings.GetStore().List() {
		config := object.(*corev1.ConfigMap)
		keys, err := indexBinding(config)
		if err != nil || len(keys) == 0 {
			continue
		}
		if len(keys) != 1 {
			continue
		}
		if _, exists := desired[keys[0]]; exists {
			ambiguous[keys[0]] = true
		}
		desired[keys[0]] = config
	}

	for key, active := range c.active {
		config := desired[key]
		if ambiguous[key] || !sameBinding(active, config) {
			delete(c.active, key)
		}
	}

	for key, config := range desired {
		if ambiguous[key] {
			continue
		}
		if _, err := c.activateBinding(key, config); err != nil {
			continue
		}
	}
	c.activated = true
}

func sameBinding(a, b *corev1.ConfigMap) bool {
	return a != nil && b != nil && a.UID == b.UID && a.Namespace == b.Namespace && a.Name == b.Name &&
		a.Labels[labels.LabelKeyAppID] == b.Labels[labels.LabelKeyAppID] &&
		a.Labels[labels.LabelKeyWorkspaceID] == b.Labels[labels.LabelKeyWorkspaceID] &&
		a.Labels[labels.LabelKeyProjectID] == b.Labels[labels.LabelKeyProjectID] &&
		a.Labels[labels.LabelKeyCallerDeploymentID] == b.Labels[labels.LabelKeyCallerDeploymentID] &&
		a.Labels[labels.LabelKeyBindingID] == b.Labels[labels.LabelKeyBindingID] &&
		a.Data["appSlug"] == b.Data["appSlug"]
}

func (c *catalog) activateBinding(key string, config *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	candidate, err := parseBinding(config)
	if err != nil {
		delete(c.active, key)
		return nil, err
	}

	active := c.active[key]
	if !sameBinding(active, config) {
		delete(c.active, key)
		active = nil
	}
	if active != nil {
		current, err := parseBinding(active)
		if err != nil {
			return nil, err
		}
		if candidate.revision == current.revision && !maps.Equal(active.Data, config.Data) {
			return nil, fmt.Errorf("binding changed without advancing revision")
		}
		if candidate.revision <= current.revision {
			return active, nil
		}
	}

	if !candidate.resolved {
		delete(c.active, key)
		return nil, fmt.Errorf("binding target is unresolved")
	}

	identity := caller{
		workspace:  config.Labels[labels.LabelKeyWorkspaceID],
		project:    config.Labels[labels.LabelKeyProjectID],
		kind:       "",
		deployment: config.Labels[labels.LabelKeyCallerDeploymentID],
		namespace:  config.Namespace,
	}
	if _, err := c.resolveBinding(identity, config); err != nil {
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

func parseBinding(config *corev1.ConfigMap) (binding, error) {
	revision, err := strconv.ParseUint(config.Data["revision"], 10, 64)
	if err != nil {
		return binding{}, fault.Wrap(err, fault.Internal("invalid binding revision"))
	}

	err = assert.All(
		assert.True(revision > 0, "binding revision must be positive"),
		assert.NotEmpty(config.UID),
		assert.True(config.DeletionTimestamp == nil, "binding is terminating"),
		assert.NotEmpty(config.Labels[labels.LabelKeyCallerDeploymentID], "caller deployment ID is required"),
		assert.NotEmpty(config.Labels[labels.LabelKeyBindingID], "binding ID is required"),
		assert.NotEmpty(config.Labels[labels.LabelKeyAppID]),
		assert.Equal(config.Data["deploymentId"] == "", config.Data["serviceName"] == "", "binding target must be entirely resolved or unresolved"),
	)
	if config.Data["serviceName"] != "" {
		err = assert.All(err, assert.Equal(len(validation.IsDNS1035Label(config.Data["serviceName"])), 0, "invalid service name"))
	}
	if err != nil {
		return binding{}, fault.Wrap(err, fault.Internal("invalid app binding"))
	}

	return binding{
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

func indexBinding(object any) ([]string, error) {
	config := object.(*corev1.ConfigMap)
	l := config.Labels
	app := config.Data["appSlug"]
	if l[labels.LabelKeyManagedBy] != "krane" || l[labels.LabelKeyComponent] != bindingComponent ||
		l[labels.LabelKeyWorkspaceID] == "" || l[labels.LabelKeyProjectID] == "" ||
		l[labels.LabelKeyCallerDeploymentID] == "" || l[labels.LabelKeyBindingID] == "" ||
		l[labels.LabelKeyAppID] == "" || app == "" || strings.ToLower(app) != app || len(validation.IsDNS1123Label(app)) != 0 {
		return nil, nil
	}

	return []string{appKey(l[labels.LabelKeyWorkspaceID], l[labels.LabelKeyProjectID], l[labels.LabelKeyCallerDeploymentID], app)}, nil
}
