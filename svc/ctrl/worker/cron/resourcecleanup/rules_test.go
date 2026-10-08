package resourcecleanup

import (
	"database/sql"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

const internalWorkspaceID = "unkey_internal"

type fixture struct {
	t           *testing.T
	h           *Handler
	db          db.Database
	workspaceID string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	cfg := containers.MySQLIsolated(t)
	database, err := db.New(cfg.DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	return &fixture{t: t, h: &Handler{db: database}, db: database, workspaceID: uid.New(uid.WorkspacePrefix)}
}

func (f *fixture) exec(query string, args ...any) uint64 {
	f.t.Helper()
	result, err := f.db.RW().ExecContext(f.t.Context(), query, args...)
	require.NoError(f.t, err, query)
	pk, err := result.LastInsertId()
	require.NoError(f.t, err)
	return uint64(pk)
}

func (f *fixture) pks(table string) []uint64 {
	f.t.Helper()
	rows, err := f.db.RW().QueryContext(f.t.Context(), "SELECT pk FROM "+table+" ORDER BY pk")
	require.NoError(f.t, err)
	defer func() { require.NoError(f.t, rows.Close()) }()
	pks := []uint64{}
	for rows.Next() {
		var pk uint64
		require.NoError(f.t, rows.Scan(&pk))
		pks = append(pks, pk)
	}
	require.NoError(f.t, rows.Err())
	return pks
}

func (f *fixture) policyIDs() []string {
	f.t.Helper()
	rows, err := f.db.RW().QueryContext(f.t.Context(), "SELECT id FROM horizontal_autoscaling_policies ORDER BY id")
	require.NoError(f.t, err)
	defer func() { require.NoError(f.t, rows.Close()) }()
	ids := []string{}
	for rows.Next() {
		var id string
		require.NoError(f.t, rows.Scan(&id))
		ids = append(ids, id)
	}
	require.NoError(f.t, rows.Err())
	return ids
}

func (f *fixture) project(id string) uint64 {
	return f.exec("INSERT INTO projects (id, workspace_id, name, slug, created_at) VALUES (?, ?, ?, ?, 1)", id, f.workspaceID, id, id)
}

func (f *fixture) app(id, projectID string) uint64 {
	return f.exec("INSERT INTO apps (id, workspace_id, project_id, name, slug, created_at) VALUES (?, ?, ?, ?, ?, 1)", id, f.workspaceID, projectID, id, id)
}

func (f *fixture) environment(id, appID string) uint64 {
	return f.exec("INSERT INTO environments (id, workspace_id, project_id, app_id, slug, created_at) VALUES (?, ?, '', ?, ?, 1)", id, f.workspaceID, appID, id)
}

func (f *fixture) deployment(id, appID string) uint64 {
	return f.exec(`INSERT INTO deployments (id, k8s_name, workspace_id, project_id, environment_id, app_id,
		sentinel_config, cpu_millicores, memory_mib, encrypted_environment_variables, created_at)
		VALUES (?, ?, ?, '', '', ?, '', 250, 256, '', 1)`, id, id, f.workspaceID, appID)
}

func (f *fixture) topology(deploymentID, regionID string) {
	f.exec("INSERT INTO deployment_topology (workspace_id, deployment_id, region_id, desired_status, created_at) VALUES (?, ?, ?, 'running', 1)", f.workspaceID, deploymentID, regionID)
}

func (f *fixture) portal(id, projectID string, appID sql.NullString) uint64 {
	return f.exec("INSERT INTO portals (id, workspace_id, project_id, slug, display_name, app_id, created_at) VALUES (?, ?, ?, ?, 'portal', ?, 1)", id, f.workspaceID, projectID, id, appID)
}

func (f *fixture) domain(id, workspaceID, environmentID, hostname string) uint64 {
	return f.exec(`INSERT INTO custom_domains (id, workspace_id, project_id, app_id, environment_id, domain,
		challenge_type, verification_token, target_cname, created_at)
		VALUES (?, ?, '', '', ?, ?, 'HTTP-01', 'token', ?, 1)`, id, workspaceID, environmentID, hostname, id)
}

func (f *fixture) challenge(workspaceID, domainID string) uint64 {
	return f.exec("INSERT INTO acme_challenges (domain_id, workspace_id, token, challenge_type, `authorization`, status, expires_at, created_at) VALUES (?, ?, 'token', 'HTTP-01', 'auth', 'pending', 1, 1)", domainID, workspaceID)
}

func (f *fixture) certificate(workspaceID, hostname string) uint64 {
	return f.exec("INSERT INTO certificates (id, workspace_id, hostname, certificate, encrypted_private_key, created_at) VALUES (?, ?, ?, '', '', 1)", uid.New(uid.CertificatePrefix), workspaceID, hostname)
}

func (f *fixture) instance(deploymentID, regionID string) uint64 {
	id := uid.New(uid.InstancePrefix)
	return f.exec(`INSERT INTO instances (id, deployment_id, workspace_id, project_id, app_id, region_id, k8s_name, address, cpu_millicores, memory_mib, status)
		VALUES (?, ?, ?, '', '', ?, ?, '', 250, 256, 'running')`, id, deploymentID, f.workspaceID, regionID, id)
}

func (f *fixture) ciliumPolicy(deploymentID string) uint64 {
	return f.exec(`INSERT INTO cilium_network_policies (id, workspace_id, project_id, app_id, environment_id, deployment_id, k8s_name, k8s_namespace, region_id, policy, created_at)
		VALUES (?, ?, '', '', '', ?, 'policy', 'ns', ?, '{}', 1)`, uid.New(uid.CiliumNetworkPolicyPrefix), f.workspaceID, deploymentID, uid.New(uid.RegionPrefix))
}

func (f *fixture) autoscalingPolicy(id string) {
	f.exec("INSERT INTO horizontal_autoscaling_policies (id, workspace_id, replicas_min, replicas_max, created_at) VALUES (?, ?, 1, 3, 1)", id, f.workspaceID)
}

func (f *fixture) regionalSetting(environmentID string, policyID sql.NullString) uint64 {
	return f.exec("INSERT INTO app_regional_settings (workspace_id, app_id, environment_id, region_id, horizontal_autoscaling_policy_id, created_at) VALUES (?, ?, ?, ?, ?, 1)", f.workspaceID, uid.New(uid.AppPrefix), environmentID, uid.New(uid.RegionPrefix), policyID)
}

func (f *fixture) openapiSpec(deploymentID, portalID sql.NullString) uint64 {
	return f.exec("INSERT INTO openapi_specs (id, workspace_id, deployment_id, portal_id, content, created_at) VALUES (?, ?, ?, ?, '', 1)", uid.New(uid.OpenApiSpecPrefix), f.workspaceID, deploymentID, portalID)
}

func valid(s string) sql.NullString {
	return sql.NullString{String: s, Valid: true}
}

type childCase struct {
	live   string
	insert func(owner string) uint64
	revive func(owner string)
}

type liveParents struct {
	projectID, appID, environmentID, deploymentID, portalID, domainID, hostname string
}

func (f *fixture) childCases(p liveParents) map[string]childCase {
	reviveProject := func(id string) { f.project(id) }
	reviveApp := func(id string) { f.app(id, p.projectID) }
	reviveEnvironment := func(id string) { f.environment(id, p.appID) }
	reviveDeployment := func(id string) { f.deployment(id, p.appID) }
	revivePortal := func(id string) { f.portal(id, p.projectID, sql.NullString{}) }
	return map[string]childCase{
		"apps": {live: p.projectID, revive: reviveProject, insert: func(owner string) uint64 {
			return f.app(uid.New(uid.AppPrefix), owner)
		}},
		"github_repo_connections": {live: p.appID, revive: reviveApp, insert: func(owner string) uint64 {
			return f.exec("INSERT INTO github_repo_connections (workspace_id, project_id, app_id, installation_id, repository_id, repository_full_name, created_at) VALUES (?, '', ?, 1, 1, 'acme/repo', 1)", f.workspaceID, owner)
		}},
		"app_source_oci": {live: p.appID, revive: reviveApp, insert: func(owner string) uint64 {
			return f.exec("INSERT INTO app_source_oci (workspace_id, app_id, image_reference, created_at) VALUES (?, ?, 'nginx', 1)", f.workspaceID, owner)
		}},
		"portals": {live: p.projectID, revive: reviveProject, insert: func(owner string) uint64 {
			return f.portal(uid.New(uid.PortalPrefix), owner, sql.NullString{})
		}},
		"portal_sessions": {live: p.portalID, revive: revivePortal, insert: func(owner string) uint64 {
			return f.exec("INSERT INTO portal_sessions (id, workspace_id, portal_id, external_id, scopes, exchange_code_hash, exchange_code_expires_at, created_at) VALUES (?, ?, ?, 'user', '[]', ?, 1, 1)", uid.New(uid.PortalSessionPrefix), f.workspaceID, owner, uid.New(uid.PortalExchangeCodePrefix))
		}},
		"openapi_specs": {live: p.deploymentID, revive: reviveDeployment, insert: func(owner string) uint64 {
			return f.openapiSpec(valid(owner), sql.NullString{})
		}},
		"frontline_routes": {live: p.deploymentID, revive: reviveDeployment, insert: func(owner string) uint64 {
			id := uid.New(uid.FrontlineRoutePrefix)
			return f.exec("INSERT INTO frontline_routes (id, project_id, app_id, deployment_id, environment_id, fully_qualified_domain_name, created_at) VALUES (?, '', '', ?, '', ?, 1)", id, owner, id)
		}},
		"cilium_network_policies": {live: p.deploymentID, revive: reviveDeployment, insert: f.ciliumPolicy},
		"instances": {live: p.deploymentID, revive: reviveDeployment, insert: func(owner string) uint64 {
			return f.instance(owner, uid.New(uid.RegionPrefix))
		}},
		"deployment_steps": {live: p.deploymentID, revive: reviveDeployment, insert: func(owner string) uint64 {
			return f.exec("INSERT INTO deployment_steps (workspace_id, project_id, environment_id, deployment_id, app_id, started_at) VALUES (?, '', '', ?, '', 1)", f.workspaceID, owner)
		}},
		"app_build_settings": {live: p.environmentID, revive: reviveEnvironment, insert: func(owner string) uint64 {
			return f.exec("INSERT INTO app_build_settings (workspace_id, app_id, environment_id, created_at) VALUES (?, ?, ?, 1)", f.workspaceID, uid.New(uid.AppPrefix), owner)
		}},
		"app_runtime_settings": {live: p.environmentID, revive: reviveEnvironment, insert: func(owner string) uint64 {
			return f.exec("INSERT INTO app_runtime_settings (workspace_id, app_id, environment_id, sentinel_config, created_at) VALUES (?, ?, ?, '', 1)", f.workspaceID, uid.New(uid.AppPrefix), owner)
		}},
		"app_environment_variables": {live: p.environmentID, revive: reviveEnvironment, insert: func(owner string) uint64 {
			return f.exec("INSERT INTO app_environment_variables (id, workspace_id, app_id, environment_id, `key`, value, type, created_at) VALUES (?, ?, ?, ?, 'KEY', 'value', 'recoverable', 1)", uid.New(uid.EnvironmentVariablePrefix), f.workspaceID, uid.New(uid.AppPrefix), owner)
		}},
		"app_regional_settings": {live: p.environmentID, revive: reviveEnvironment, insert: func(owner string) uint64 {
			return f.regionalSetting(owner, sql.NullString{})
		}},
		"custom_domains": {live: p.environmentID, revive: reviveEnvironment, insert: func(owner string) uint64 {
			id := uid.New(uid.DomainPrefix)
			return f.domain(id, f.workspaceID, owner, id+".example.com")
		}},
		"acme_challenges": {live: p.domainID, insert: func(owner string) uint64 { return f.challenge(f.workspaceID, owner) }, revive: func(owner string) {
			f.domain(owner, f.workspaceID, p.environmentID, owner+".example.com")
		}},
		"certificates": {live: p.hostname, insert: func(owner string) uint64 { return f.certificate(f.workspaceID, owner) }, revive: func(owner string) {
			f.domain(uid.New(uid.DomainPrefix), f.workspaceID, p.environmentID, owner)
		}},
	}
}

func TestCleanupRulesReclaimOnlyOrphans(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	ctx := t.Context()
	p := liveParents{
		projectID:     uid.New(uid.ProjectPrefix),
		appID:         uid.New(uid.AppPrefix),
		environmentID: uid.New(uid.EnvironmentPrefix),
		deploymentID:  uid.New(uid.DeploymentPrefix),
		portalID:      uid.New(uid.PortalPrefix),
		domainID:      uid.New(uid.DomainPrefix),
		hostname:      "live.example.com",
	}
	kept := map[string][]uint64{}
	keep := func(table string, pk uint64) { kept[table] = append(kept[table], pk) }
	f.project(p.projectID)
	keep("apps", f.app(p.appID, p.projectID))
	f.environment(p.environmentID, p.appID)
	f.deployment(p.deploymentID, p.appID)
	keep("portals", f.portal(p.portalID, p.projectID, sql.NullString{}))
	keep("custom_domains", f.domain(p.domainID, f.workspaceID, p.environmentID, p.hostname))

	cases := f.childCases(p)
	require.Len(t, cleanupRules, 17)
	for _, rule := range cleanupRules {
		c, ok := cases[rule.name]
		require.True(t, ok, "missing case for rule %s", rule.name)
		keep(rule.name, c.insert(c.live))
		c.insert("dead_" + uid.New(uid.TestPrefix))
	}

	appWithEnvironment, appWithDeployment := uid.New(uid.AppPrefix), uid.New(uid.AppPrefix)
	keep("apps", f.app(appWithEnvironment, "dead_project"))
	f.environment(uid.New(uid.EnvironmentPrefix), appWithEnvironment)
	keep("apps", f.app(appWithDeployment, "dead_project"))
	f.deployment(uid.New(uid.DeploymentPrefix), appWithDeployment)

	f.portal(uid.New(uid.PortalPrefix), p.projectID, valid("dead_app"))
	keep("portals", f.portal(uid.New(uid.PortalPrefix), p.projectID, valid(p.appID)))
	f.openapiSpec(sql.NullString{}, valid("dead_portal"))
	keep("openapi_specs", f.openapiSpec(sql.NullString{}, valid(p.portalID)))

	protectedDeployment, regionID := uid.New(uid.DeploymentPrefix), uid.New(uid.RegionPrefix)
	f.topology(protectedDeployment, regionID)
	keep("cilium_network_policies", f.ciliumPolicy(protectedDeployment))
	keep("instances", f.instance(protectedDeployment, regionID))
	f.instance(protectedDeployment, uid.New(uid.RegionPrefix))

	sharedPolicy, privatePolicy := uid.New(uid.AutoscalingPolicyPrefix), uid.New(uid.AutoscalingPolicyPrefix)
	f.autoscalingPolicy(sharedPolicy)
	f.autoscalingPolicy(privatePolicy)
	f.regionalSetting("dead_environment", valid(sharedPolicy))
	keep("app_regional_settings", f.regionalSetting(p.environmentID, valid(sharedPolicy)))
	f.regionalSetting("dead_environment", valid(privatePolicy))
	f.regionalSetting("dead_environment", valid(privatePolicy))

	keep("custom_domains", f.domain(uid.New(uid.DomainPrefix), internalWorkspaceID, "dead_environment", "internal.example.com"))
	keep("acme_challenges", f.challenge(internalWorkspaceID, "dead_domain"))
	keep("certificates", f.certificate(internalWorkspaceID, "orphan-internal.example.com"))

	otherWorkspaceID := uid.New(uid.WorkspacePrefix)
	keep("custom_domains", f.domain(uid.New(uid.DomainPrefix), f.workspaceID, p.environmentID, "shared.example.com"))
	keep("custom_domains", f.domain(uid.New(uid.DomainPrefix), otherWorkspaceID, p.environmentID, "shared.example.com"))
	keep("certificates", f.certificate(otherWorkspaceID, "shared.example.com"))
	keep("custom_domains", f.domain(uid.New(uid.DomainPrefix), f.workspaceID, p.environmentID, "foreign.example.com"))
	f.certificate(otherWorkspaceID, "foreign.example.com")

	for _, rule := range cleanupRules {
		page, err := f.h.cleanPage(ctx, rule, 0)
		require.NoError(t, err, rule.name)
		require.Less(t, page.Scanned, batchSize, rule.name)
	}
	for _, rule := range cleanupRules {
		require.ElementsMatch(t, kept[rule.name], f.pks(rule.name), "rule %s", rule.name)
	}
	require.Equal(t, []string{sharedPolicy}, f.policyIDs())

	for _, rule := range cleanupRules {
		c := cases[rule.name]
		owner := "late_" + uid.New(uid.TestPrefix)
		pk := c.insert(owner)
		c.revive(owner)
		result, err := f.db.RW().ExecContext(ctx, rule.remove, pk-1, pk)
		require.NoError(t, err, rule.name)
		deleted, err := result.RowsAffected()
		require.NoError(t, err)
		require.Zero(t, deleted, "rule %s deleted pk %d after its owner %s was recreated", rule.name, pk, owner)
		require.Contains(t, f.pks(rule.name), pk, rule.name)
	}
}

func TestCleanPagePaginatesSharedAutoscalingPolicies(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	ctx := t.Context()
	rule := cleanupRules[slices.IndexFunc(cleanupRules, func(r cleanupRule) bool { return r.name == "app_regional_settings" })]
	appID, environmentID := uid.New(uid.AppPrefix), uid.New(uid.EnvironmentPrefix)
	f.environment(environmentID, appID)
	orphanedPolicy, livePolicy := uid.New(uid.AutoscalingPolicyPrefix), uid.New(uid.AutoscalingPolicyPrefix)
	f.autoscalingPolicy(orphanedPolicy)
	f.autoscalingPolicy(livePolicy)

	f.regionalSetting("dead_environment", valid(orphanedPolicy))
	f.regionalSetting("dead_environment", valid(livePolicy))
	values := make([]string, 0, batchSize-2)
	args := make([]any, 0, 2*(batchSize-2))
	for range batchSize - 2 {
		values = append(values, "(?, 'app', 'dead_environment', ?, 1)")
		args = append(args, f.workspaceID, uid.New(uid.RegionPrefix))
	}
	f.exec("INSERT INTO app_regional_settings (workspace_id, app_id, environment_id, region_id, created_at) VALUES "+strings.Join(values, ","), args...)
	firstPage := f.pks("app_regional_settings")
	require.Len(t, firstPage, batchSize)
	f.regionalSetting("dead_environment", valid(orphanedPolicy))
	liveSetting := f.regionalSetting(environmentID, valid(livePolicy))

	page, err := f.h.cleanPage(ctx, rule, 0)
	require.NoError(t, err)
	require.Equal(t, cleanupPage{AfterPK: firstPage[batchSize-1], Scanned: batchSize, Deleted: batchSize}, page)
	require.ElementsMatch(t, []string{orphanedPolicy, livePolicy}, f.policyIDs(), "policy referenced beyond the page boundary must survive")

	page, err = f.h.cleanPage(ctx, rule, page.AfterPK)
	require.NoError(t, err)
	require.Equal(t, cleanupPage{AfterPK: liveSetting, Scanned: 2, Deleted: 2}, page)
	require.Equal(t, []uint64{liveSetting}, f.pks("app_regional_settings"))
	require.Equal(t, []string{livePolicy}, f.policyIDs())

	page, err = f.h.cleanPage(ctx, rule, page.AfterPK)
	require.NoError(t, err)
	require.Equal(t, cleanupPage{AfterPK: liveSetting}, page)

	f.exec("DELETE FROM environments WHERE id = ?", environmentID)
	page, err = f.h.cleanPage(ctx, rule, 0)
	require.NoError(t, err)
	require.Equal(t, cleanupPage{AfterPK: liveSetting, Scanned: 1, Deleted: 2}, page)
	require.Empty(t, f.pks("app_regional_settings"))
	require.Empty(t, f.policyIDs())
}
