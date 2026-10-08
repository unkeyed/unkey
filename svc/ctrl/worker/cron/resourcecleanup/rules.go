package resourcecleanup

type cleanupRule struct {
	name   string
	scan   string
	remove string
}

var cleanupRules = []cleanupRule{
	{
		name: "apps",
		scan: "SELECT pk FROM apps WHERE pk > ? ORDER BY pk LIMIT ?",
		remove: `DELETE FROM apps WHERE pk > ? AND pk <= ?
			AND NOT EXISTS (SELECT 1 FROM projects p WHERE p.id = apps.project_id)
			AND NOT EXISTS (SELECT 1 FROM environments e WHERE e.app_id = apps.id)
			AND NOT EXISTS (SELECT 1 FROM deployments d WHERE d.app_id = apps.id)`,
	},
	{
		name: "github_repo_connections",
		scan: "SELECT pk FROM github_repo_connections WHERE pk > ? ORDER BY pk LIMIT ?",
		remove: `DELETE FROM github_repo_connections WHERE pk > ? AND pk <= ?
			AND NOT EXISTS (SELECT 1 FROM apps a WHERE a.id = github_repo_connections.app_id)`,
	},
	{
		name: "app_source_oci",
		scan: "SELECT pk FROM app_source_oci WHERE pk > ? ORDER BY pk LIMIT ?",
		remove: `DELETE FROM app_source_oci WHERE pk > ? AND pk <= ?
			AND NOT EXISTS (SELECT 1 FROM apps a WHERE a.id = app_source_oci.app_id)`,
	},
	{
		name: "portals",
		scan: "SELECT pk FROM portals WHERE pk > ? ORDER BY pk LIMIT ?",
		remove: `DELETE FROM portals WHERE pk > ? AND pk <= ? AND (
			NOT EXISTS (SELECT 1 FROM projects p WHERE p.id = portals.project_id)
			OR (app_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM apps a WHERE a.id = portals.app_id)))`,
	},
	{
		name: "portal_sessions",
		scan: "SELECT pk FROM portal_sessions WHERE pk > ? ORDER BY pk LIMIT ?",
		remove: `DELETE FROM portal_sessions WHERE pk > ? AND pk <= ?
			AND NOT EXISTS (SELECT 1 FROM portals p WHERE p.id = portal_sessions.portal_id)`,
	},
	{
		name: "openapi_specs",
		scan: "SELECT pk FROM openapi_specs WHERE pk > ? ORDER BY pk LIMIT ?",
		remove: `DELETE FROM openapi_specs WHERE pk > ? AND pk <= ? AND (
			(deployment_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM deployments d WHERE d.id = openapi_specs.deployment_id))
			OR (portal_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM portals p WHERE p.id = openapi_specs.portal_id)))`,
	},
	{
		name: "frontline_routes",
		scan: "SELECT pk FROM frontline_routes WHERE pk > ? ORDER BY pk LIMIT ?",
		remove: `DELETE FROM frontline_routes WHERE pk > ? AND pk <= ?
			AND NOT EXISTS (SELECT 1 FROM deployments d WHERE d.id = frontline_routes.deployment_id)`,
	},
	{
		name: "cilium_network_policies",
		scan: "SELECT pk FROM cilium_network_policies WHERE pk > ? ORDER BY pk LIMIT ?",
		remove: `DELETE FROM cilium_network_policies WHERE pk > ? AND pk <= ?
			AND NOT EXISTS (SELECT 1 FROM deployments d WHERE d.id = cilium_network_policies.deployment_id)
			AND NOT EXISTS (SELECT 1 FROM deployment_topology dt WHERE dt.deployment_id = cilium_network_policies.deployment_id)`,
	},
	{
		name: "instances",
		scan: "SELECT pk FROM instances WHERE pk > ? ORDER BY pk LIMIT ?",
		remove: `DELETE FROM instances WHERE pk > ? AND pk <= ?
			AND NOT EXISTS (SELECT 1 FROM deployments d WHERE d.id = instances.deployment_id)
			AND NOT EXISTS (SELECT 1 FROM deployment_topology dt WHERE dt.deployment_id = instances.deployment_id AND dt.region_id = instances.region_id)`,
	},
	{
		name: "deployment_steps",
		scan: "SELECT pk FROM deployment_steps WHERE pk > ? ORDER BY pk LIMIT ?",
		remove: `DELETE FROM deployment_steps WHERE pk > ? AND pk <= ?
			AND NOT EXISTS (SELECT 1 FROM deployments d WHERE d.id = deployment_steps.deployment_id)`,
	},
	{
		name: "app_build_settings",
		scan: "SELECT pk FROM app_build_settings WHERE pk > ? ORDER BY pk LIMIT ?",
		remove: `DELETE FROM app_build_settings WHERE pk > ? AND pk <= ?
			AND NOT EXISTS (SELECT 1 FROM environments e WHERE e.id = app_build_settings.environment_id)`,
	},
	{
		name: "app_runtime_settings",
		scan: "SELECT pk FROM app_runtime_settings WHERE pk > ? ORDER BY pk LIMIT ?",
		remove: `DELETE FROM app_runtime_settings WHERE pk > ? AND pk <= ?
			AND NOT EXISTS (SELECT 1 FROM environments e WHERE e.id = app_runtime_settings.environment_id)`,
	},
	{
		name: "app_environment_variables",
		scan: "SELECT pk FROM app_environment_variables WHERE pk > ? ORDER BY pk LIMIT ?",
		remove: `DELETE FROM app_environment_variables WHERE pk > ? AND pk <= ?
			AND NOT EXISTS (SELECT 1 FROM environments e WHERE e.id = app_environment_variables.environment_id)`,
	},
	{
		name: "app_regional_settings",
		scan: "SELECT pk FROM app_regional_settings WHERE pk > ? ORDER BY pk LIMIT ?",
		remove: `DELETE s, p FROM (SELECT ? AS after_pk, ? AS through_pk) bounds
			JOIN app_regional_settings s ON s.pk > bounds.after_pk AND s.pk <= bounds.through_pk
			LEFT JOIN app_regional_settings other ON other.horizontal_autoscaling_policy_id = s.horizontal_autoscaling_policy_id
				AND (other.pk <= bounds.after_pk OR other.pk > bounds.through_pk
					OR EXISTS (SELECT 1 FROM environments e WHERE e.id = other.environment_id))
			LEFT JOIN horizontal_autoscaling_policies p ON p.id = s.horizontal_autoscaling_policy_id AND other.pk IS NULL
			WHERE NOT EXISTS (SELECT 1 FROM environments e WHERE e.id = s.environment_id)`,
	},
	{
		name: "custom_domains",
		scan: "SELECT pk FROM custom_domains WHERE pk > ? ORDER BY pk LIMIT ?",
		remove: `DELETE FROM custom_domains WHERE pk > ? AND pk <= ? AND workspace_id <> 'unkey_internal'
			AND NOT EXISTS (SELECT 1 FROM environments e WHERE e.id = custom_domains.environment_id)`,
	},
	{
		name: "acme_challenges",
		scan: "SELECT pk FROM acme_challenges WHERE pk > ? ORDER BY pk LIMIT ?",
		remove: `DELETE FROM acme_challenges WHERE pk > ? AND pk <= ? AND workspace_id <> 'unkey_internal'
			AND NOT EXISTS (SELECT 1 FROM custom_domains d WHERE d.id = acme_challenges.domain_id)`,
	},
	{
		name: "certificates",
		scan: "SELECT pk FROM certificates WHERE pk > ? ORDER BY pk LIMIT ?",
		remove: `DELETE FROM certificates WHERE pk > ? AND pk <= ? AND workspace_id <> 'unkey_internal'
			AND NOT EXISTS (SELECT 1 FROM custom_domains d WHERE d.domain = certificates.hostname AND d.workspace_id = certificates.workspace_id)`,
	},
}
