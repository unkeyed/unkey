package discovery

import (
	"net/netip"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/deploy/privatenetwork"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

type connection struct {
	appID      string
	deployment string
	service    string
	namespace  string
	resolved   bool
}

// Resolve returns the ready private IPv4 endpoints, in ascending order, that
// name gives caller. name is an alias, or <selector>.<alias> where selector is
// a region or local-first. It reports false if caller has no connection for
// the alias or the selector is invalid. A known alias that can't be answered
// returns true and an [*Error] whose reason names the failure; a cache failure
// returns an error with reason lookup_error.
func (c *Catalog) Resolve(caller Caller, name string) ([]netip.Addr, bool, error) {
	app := name
	selector := ""
	if prefix, name, selected := strings.Cut(app, "."); selected {
		if len(validation.IsDNS1123Label(prefix)) != 0 || len(validation.IsDNS1123Label(name)) != 0 {
			return nil, false, nil
		}
		selector, app = prefix, name
	}

	config, exists, err := c.lookupConnection(caller, app)
	if err != nil || !exists {
		return nil, exists, err
	}

	addresses, err := c.resolveConnection(caller, config, selector)
	return addresses, true, err
}

func (c *Catalog) lookupConnection(caller Caller, app string) (*corev1.ConfigMap, bool, error) {
	if caller.Deployment == "" {
		return nil, false, nil
	}

	key := appKey(caller.Workspace, caller.Project, caller.Deployment, app)
	objects, err := c.connections.GetIndexer().ByIndex(appIndex, key)
	if err != nil {
		return nil, false, err
	}
	if len(objects) == 0 {
		return nil, false, nil
	}
	if len(objects) != 1 {
		return nil, true, fail(ReasonConnectionAmbiguous, "ambiguous app connection")
	}

	config := objects[0].(*corev1.ConfigMap)
	connection, err := parseConnection(config)
	if err != nil {
		return nil, true, err
	}
	if !connection.resolved {
		return nil, true, fail(ReasonConnectionUnresolved, "connection target is unresolved")
	}
	return config, true, nil
}

func (c *Catalog) updateStatuses() time.Time {
	retirement := c.nextRetirement()
	published := make(map[string][]*corev1.ConfigMap)
	statuses := make(map[string]connectionStatus)
	for _, object := range c.connections.GetStore().List() {
		config := object.(*corev1.ConfigMap)
		keys, err := indexConnection(config)
		if err != nil || len(keys) != 1 {
			statuses[config.Namespace+"/"+config.Name] = newConnectionStatus(config, ReasonConnectionInvalid, fail(ReasonConnectionInvalid, "invalid app connection"))
			continue
		}
		published[keys[0]] = append(published[keys[0]], config)
	}

	for _, configs := range published {
		if len(configs) != 1 {
			for _, config := range configs {
				statuses[config.Namespace+"/"+config.Name] = newConnectionStatus(config, ReasonConnectionAmbiguous, fail(ReasonConnectionAmbiguous, "ambiguous app connection"))
			}
			continue
		}
		config := configs[0]
		status := newConnectionStatus(config, stateActive, nil)
		connection, err := parseConnection(config)
		if err != nil {
			status = newConnectionStatus(config, ReasonOf(err), err)
		} else if !connection.resolved {
			status = newConnectionStatus(config, ReasonConnectionUnresolved, fail(ReasonConnectionUnresolved, "connection target is unresolved"))
		} else if _, err := c.resolveConnection(connectionCaller(config), config, ""); err != nil {
			status = newConnectionStatus(config, ReasonOf(err), err)
		}
		statuses[config.Namespace+"/"+config.Name] = status
	}

	counts := map[string]map[Reason]int{kindConnection: {}, kindReplica: {}}
	for _, status := range statuses {
		counts[status.kind][status.state]++
	}

	c.statusMu.Lock()
	changes := connectionChanges(c.statuses, statuses)
	c.statuses = statuses
	reportConnections(counts)
	c.statusMu.Unlock()

	for _, change := range changes {
		change.log()
	}
	return retirement
}

func (c *Catalog) nextRetirement() time.Time {
	now := c.clock.Now()
	var next time.Time
	for _, object := range c.services.GetStore().List() {
		service := object.(*corev1.Service)
		deadline, err := time.Parse(time.RFC3339Nano, service.Annotations[privatenetwork.RetireAfterAnnotation])
		if err != nil || !deadline.After(now) {
			continue
		}
		if next.IsZero() || deadline.Before(next) {
			next = deadline
		}
	}
	return next
}

func connectionCaller(config *corev1.ConfigMap) Caller {
	return Caller{
		Workspace:  config.Labels[privatenetwork.WorkspaceLabel],
		Project:    config.Labels[privatenetwork.ProjectLabel],
		Deployment: config.Labels[privatenetwork.CallerDeploymentLabel],
		Namespace:  config.Namespace,
	}
}

func parseConnection(config *corev1.ConfigMap) (connection, error) {
	data, err := privatenetwork.Decode(config.Data)
	if err != nil {
		return connection{}, fail(ReasonConnectionInvalid, "invalid app connection: %w", err)
	}

	err = assert.All(
		assert.NotEmpty(config.UID),
		assert.True(config.DeletionTimestamp == nil, "connection is terminating"),
		assert.NotEmpty(config.Labels[privatenetwork.CallerDeploymentLabel], "caller deployment ID is required"),
		assert.NotEmpty(config.Labels[privatenetwork.ConnectionLabel], "connection ID is required"),
		assert.NotEmpty(config.Labels[privatenetwork.AppLabel]),
	)
	if err != nil {
		return connection{}, fail(ReasonConnectionInvalid, "invalid app connection: %w", err)
	}

	return connection{
		appID:      config.Labels[privatenetwork.AppLabel],
		deployment: data.DeploymentID,
		service:    data.ServiceName,
		namespace:  config.Namespace,
		resolved:   data.DeploymentID != "",
	}, nil
}

func appKey(workspace, project, callerDeployment, app string) string {
	return workspace + "/" + project + "/" + callerDeployment + "/" + app
}

func indexConnection(object any) ([]string, error) {
	config := object.(*corev1.ConfigMap)
	l := config.Labels
	app := config.Data[privatenetwork.ConnectionAliasKey]
	if l[privatenetwork.ManagedByLabel] != "krane" || l[privatenetwork.ComponentLabel] != privatenetwork.DiscoveryComponent ||
		l[privatenetwork.WorkspaceLabel] == "" || l[privatenetwork.ProjectLabel] == "" ||
		l[privatenetwork.CallerDeploymentLabel] == "" || l[privatenetwork.ConnectionLabel] == "" ||
		l[privatenetwork.AppLabel] == "" || app == "" || strings.ToLower(app) != app || len(validation.IsDNS1123Label(app)) != 0 {
		return nil, nil
	}

	return []string{appKey(l[privatenetwork.WorkspaceLabel], l[privatenetwork.ProjectLabel], l[privatenetwork.CallerDeploymentLabel], app)}, nil
}
