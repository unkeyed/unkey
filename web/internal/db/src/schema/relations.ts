import { defineRelations } from "drizzle-orm";
import * as tables from "./generated/tables";

export const relations = defineRelations(tables, (r) => ({
  workspaces: {
    apis: r.many.apis(),
    keys: r.many.keys({
      alias: "workspace_key_relation",
    }),
    vercelIntegrations: r.many.vercelIntegrations({
      alias: "vercel_workspace_relation",
    }),
    vercelBindings: r.many.vercelBindings({
      alias: "vercel_key_binding_relation",
    }),
    roles: r.many.roles(),
    permissions: r.many.permissions(),
    ratelimitNamespaces: r.many.ratelimitNamespaces(),
    keySpaces: r.many.keyAuth(),
    identities: r.many.identities(),
    githubAppInstallations: r.many.githubAppInstallations(),
    quota: r.one.quota(),
    clickhouseSettings: r.one.clickhouseWorkspaceSettings(),
    projects: r.many.projects(),
    sentinels: r.many.sentinels(),
    certificates: r.many.certificates(),
  },

  apis: {
    workspace: r.one.workspaces({
      from: r.apis.workspaceId,
      to: r.workspaces.id,
    }),
    keyAuth: r.one.keyAuth({
      from: r.apis.keyAuthId,
      to: r.keyAuth.id,
    }),
  },

  keyAuth: {
    workspace: r.one.workspaces({
      from: r.keyAuth.workspaceId,
      to: r.workspaces.id,
    }),
    api: r.one.apis({
      from: r.keyAuth.id,
      to: r.apis.keyAuthId,
    }),
    keys: r.many.keys(),
  },

  keys: {
    keyAuth: r.one.keyAuth({
      from: r.keys.keyAuthId,
      to: r.keyAuth.id,
    }),
    workspace: r.one.workspaces({
      from: r.keys.workspaceId,
      to: r.workspaces.id,
      alias: "workspace_key_relation",
    }),
    forWorkspace: r.one.workspaces({
      from: r.keys.forWorkspaceId,
      to: r.workspaces.id,
    }),
    permissions: r.many.keysPermissions({
      alias: "keys_keys_permissions_relations",
    }),
    roles: r.many.keysRoles({
      alias: "keys_roles_key_relations",
    }),
    encrypted: r.one.encryptedKeys(),
    ratelimits: r.many.ratelimits(),
    identity: r.one.identities({
      from: r.keys.identityId,
      to: r.identities.id,
    }),
  },

  encryptedKeys: {
    key: r.one.keys({
      from: r.encryptedKeys.keyId,
      to: r.keys.id,
    }),
    workspace: r.one.workspaces({
      from: r.encryptedKeys.workspaceId,
      to: r.workspaces.id,
    }),
  },

  permissions: {
    workspace: r.one.workspaces({
      from: r.permissions.workspaceId,
      to: r.workspaces.id,
    }),
    keys: r.many.keysPermissions({
      alias: "permissions_keys_permissions_relations",
    }),
    roles: r.many.rolesPermissions({
      alias: "roles_permissions",
    }),
  },

  keysPermissions: {
    key: r.one.keys({
      from: r.keysPermissions.keyId,
      to: r.keys.id,
      alias: "keys_keys_permissions_relations",
    }),
    permission: r.one.permissions({
      from: r.keysPermissions.permissionId,
      to: r.permissions.id,
      alias: "permissions_keys_permissions_relations",
    }),
  },

  roles: {
    workspace: r.one.workspaces({
      from: r.roles.workspaceId,
      to: r.workspaces.id,
    }),
    keys: r.many.keysRoles({
      alias: "keys_roles_roles_relations",
    }),
    permissions: r.many.rolesPermissions({
      alias: "roles_rolesPermissions",
    }),
  },

  rolesPermissions: {
    role: r.one.roles({
      from: r.rolesPermissions.roleId,
      to: r.roles.id,
      alias: "roles_rolesPermissions",
    }),
    permission: r.one.permissions({
      from: r.rolesPermissions.permissionId,
      to: r.permissions.id,
      alias: "roles_permissions",
    }),
  },

  keysRoles: {
    role: r.one.roles({
      from: r.keysRoles.roleId,
      to: r.roles.id,
      alias: "keys_roles_roles_relations",
    }),
    key: r.one.keys({
      from: r.keysRoles.keyId,
      to: r.keys.id,
      alias: "keys_roles_key_relations",
    }),
  },

  identities: {
    workspace: r.one.workspaces({
      from: r.identities.workspaceId,
      to: r.workspaces.id,
    }),
    keys: r.many.keys(),
    ratelimits: r.many.ratelimits(),
  },

  ratelimits: {
    workspace: r.one.workspaces({
      from: r.ratelimits.workspaceId,
      to: r.workspaces.id,
    }),
    keys: r.one.keys({
      from: r.ratelimits.keyId,
      to: r.keys.id,
    }),
    identities: r.one.identities({
      from: r.ratelimits.identityId,
      to: r.identities.id,
    }),
  },

  ratelimitNamespaces: {
    workspace: r.one.workspaces({
      from: r.ratelimitNamespaces.workspaceId,
      to: r.workspaces.id,
    }),
    overrides: r.many.ratelimitOverrides(),
  },

  ratelimitOverrides: {
    workspace: r.one.workspaces({
      from: r.ratelimitOverrides.workspaceId,
      to: r.workspaces.id,
    }),
    namespace: r.one.ratelimitNamespaces({
      from: r.ratelimitOverrides.namespaceId,
      to: r.ratelimitNamespaces.id,
    }),
  },

  vercelIntegrations: {
    workspace: r.one.workspaces({
      from: r.vercelIntegrations.workspaceId,
      to: r.workspaces.id,
      alias: "vercel_workspace_relation",
    }),
    vercelBindings: r.many.vercelBindings(),
  },

  vercelBindings: {
    workspace: r.one.workspaces({
      from: r.vercelBindings.workspaceId,
      to: r.workspaces.id,
      alias: "vercel_key_binding_relation",
    }),
    vercelIntegrations: r.one.vercelIntegrations({
      from: r.vercelBindings.integrationId,
      to: r.vercelIntegrations.id,
    }),
  },

  quota: {
    workspace: r.one.workspaces({
      from: r.quota.workspaceId,
      to: r.workspaces.id,
    }),
  },

  clickhouseWorkspaceSettings: {
    workspace: r.one.workspaces({
      from: r.clickhouseWorkspaceSettings.workspaceId,
      to: r.workspaces.id,
    }),
  },

  auditLog: {
    workspace: r.one.workspaces({
      from: r.auditLog.workspaceId,
      to: r.workspaces.id,
    }),
    targets: r.many.auditLogTarget(),
  },

  auditLogTarget: {
    workspace: r.one.workspaces({
      from: r.auditLogTarget.workspaceId,
      to: r.workspaces.id,
    }),
    log: r.one.auditLog({
      from: r.auditLogTarget.auditLogId,
      to: r.auditLog.id,
    }),
  },

  projects: {
    workspace: r.one.workspaces({
      from: r.projects.workspaceId,
      to: r.workspaces.id,
    }),
    environments: r.many.environments(),
    apps: r.many.apps(),
    deployments: r.many.deployments(),
    frontlineRoutes: r.many.frontlineRoutes(),
    githubRepoConnections: r.many.githubRepoConnections(),
  },

  apps: {
    workspace: r.one.workspaces({
      from: r.apps.workspaceId,
      to: r.workspaces.id,
    }),
    project: r.one.projects({
      from: r.apps.projectId,
      to: r.projects.id,
    }),
    environments: r.many.environments(),
    githubRepoConnection: r.one.githubRepoConnections({
      from: r.apps.id,
      to: r.githubRepoConnections.appId,
    }),
  },

  environments: {
    workspace: r.one.workspaces({
      from: r.environments.workspaceId,
      to: r.workspaces.id,
    }),
    project: r.one.projects({
      from: r.environments.projectId,
      to: r.projects.id,
    }),
    app: r.one.apps({
      from: r.environments.appId,
      to: r.apps.id,
    }),
  },

  appBuildSettings: {
    workspace: r.one.workspaces({
      from: r.appBuildSettings.workspaceId,
      to: r.workspaces.id,
    }),
    app: r.one.apps({
      from: r.appBuildSettings.appId,
      to: r.apps.id,
    }),
    environment: r.one.environments({
      from: r.appBuildSettings.environmentId,
      to: r.environments.id,
    }),
  },

  appRuntimeSettings: {
    workspace: r.one.workspaces({
      from: r.appRuntimeSettings.workspaceId,
      to: r.workspaces.id,
    }),
    app: r.one.apps({
      from: r.appRuntimeSettings.appId,
      to: r.apps.id,
    }),
    environment: r.one.environments({
      from: r.appRuntimeSettings.environmentId,
      to: r.environments.id,
    }),
  },

  appRegionalSettings: {
    workspace: r.one.workspaces({
      from: r.appRegionalSettings.workspaceId,
      to: r.workspaces.id,
    }),
    app: r.one.apps({
      from: r.appRegionalSettings.appId,
      to: r.apps.id,
    }),
    environment: r.one.environments({
      from: r.appRegionalSettings.environmentId,
      to: r.environments.id,
    }),
    region: r.one.regions({
      from: r.appRegionalSettings.regionId,
      to: r.regions.id,
    }),
  },

  appEnvironmentVariables: {
    workspace: r.one.workspaces({
      from: r.appEnvironmentVariables.workspaceId,
      to: r.workspaces.id,
    }),
    app: r.one.apps({
      from: r.appEnvironmentVariables.appId,
      to: r.apps.id,
    }),
    environment: r.one.environments({
      from: r.appEnvironmentVariables.environmentId,
      to: r.environments.id,
    }),
  },

  deployments: {
    workspace: r.one.workspaces({
      from: r.deployments.workspaceId,
      to: r.workspaces.id,
    }),
    environment: r.one.environments({
      from: r.deployments.environmentId,
      to: r.environments.id,
    }),
    project: r.one.projects({
      from: r.deployments.projectId,
      to: r.projects.id,
    }),
    openapiSpec: r.one.openapiSpecs({
      from: r.deployments.id,
      to: r.openapiSpecs.deploymentId,
    }),
    instances: r.many.instances(),
    steps: r.many.deploymentSteps(),
  },

  deploymentSteps: {
    workspace: r.one.workspaces({
      from: r.deploymentSteps.workspaceId,
      to: r.workspaces.id,
    }),
    environment: r.one.environments({
      from: r.deploymentSteps.environmentId,
      to: r.environments.id,
    }),
    project: r.one.projects({
      from: r.deploymentSteps.projectId,
      to: r.projects.id,
    }),
    deployment: r.one.deployments({
      from: r.deploymentSteps.deploymentId,
      to: r.deployments.id,
    }),
  },

  deploymentTopology: {
    workspace: r.one.workspaces({
      from: r.deploymentTopology.workspaceId,
      to: r.workspaces.id,
    }),
    deployment: r.one.deployments({
      from: r.deploymentTopology.deploymentId,
      to: r.deployments.id,
    }),
  },

  openapiSpecs: {
    workspace: r.one.workspaces({
      from: r.openapiSpecs.workspaceId,
      to: r.workspaces.id,
    }),
    deployment: r.one.deployments({
      from: r.openapiSpecs.deploymentId,
      to: r.deployments.id,
    }),
  },

  sentinels: {
    workspace: r.one.workspaces({
      from: r.sentinels.workspaceId,
      to: r.workspaces.id,
    }),
    environment: r.one.environments({
      from: r.sentinels.environmentId,
      to: r.environments.id,
    }),
    region: r.one.regions({
      from: r.sentinels.regionId,
      to: r.regions.id,
    }),
  },

  instances: {
    deployment: r.one.deployments({
      from: r.instances.deploymentId,
      to: r.deployments.id,
    }),
    project: r.one.projects({
      from: r.instances.projectId,
      to: r.projects.id,
    }),
    region: r.one.regions({
      from: r.instances.regionId,
      to: r.regions.id,
    }),
  },

  certificates: {
    workspace: r.one.workspaces({
      from: r.certificates.workspaceId,
      to: r.workspaces.id,
    }),
  },

  frontlineRoutes: {
    deployment: r.one.deployments({
      from: r.frontlineRoutes.deploymentId,
      to: r.deployments.id,
    }),
    project: r.one.projects({
      from: r.frontlineRoutes.projectId,
      to: r.projects.id,
    }),
  },

  acmeChallenges: {
    workspace: r.one.workspaces({
      from: r.acmeChallenges.workspaceId,
      to: r.workspaces.id,
    }),
    domain: r.one.customDomains({
      from: r.acmeChallenges.domainId,
      to: r.customDomains.id,
    }),
  },

  githubAppInstallations: {
    workspace: r.one.workspaces({
      from: r.githubAppInstallations.workspaceId,
      to: r.workspaces.id,
    }),
  },

  githubRepoConnections: {
    project: r.one.projects({
      from: r.githubRepoConnections.projectId,
      to: r.projects.id,
    }),
    app: r.one.apps({
      from: r.githubRepoConnections.appId,
      to: r.apps.id,
    }),
    installation: r.one.githubAppInstallations({
      from: r.githubRepoConnections.installationId,
      to: r.githubAppInstallations.installationId,
    }),
  },

  ciliumNetworkPolicies: {
    workspace: r.one.workspaces({
      from: r.ciliumNetworkPolicies.workspaceId,
      to: r.workspaces.id,
    }),
    environment: r.one.environments({
      from: r.ciliumNetworkPolicies.environmentId,
      to: r.environments.id,
    }),
    deployment: r.one.deployments({
      from: r.ciliumNetworkPolicies.deploymentId,
      to: r.deployments.id,
    }),
  },

  clusters: {
    region: r.one.regions({
      from: r.clusters.regionId,
      to: r.regions.id,
    }),
  },

  regions: {
    clusters: r.many.clusters(),
  },

  horizontalAutoscalingPolicies: {
    workspace: r.one.workspaces({
      from: r.horizontalAutoscalingPolicies.workspaceId,
      to: r.workspaces.id,
    }),
  },
}));
