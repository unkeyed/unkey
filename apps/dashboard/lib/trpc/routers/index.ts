import { t } from "../trpc";
import { createApi } from "./api/create";
import { deleteApi } from "./api/delete";
import { keysLlmSearch } from "./api/keys/llm-search";
import { apiKeysLlmSearch } from "./api/keys/llm-search-api-keys";
import { activeKeysTimeseries } from "./api/keys/query-active-keys-timeseries";
import { queryKeysList } from "./api/keys/query-api-keys";
import { keyUsageTimeseries } from "./api/keys/query-key-usage-timeseries";
import { keyLastVerificationTime } from "./api/keys/query-latest-verification";
import { queryKeysOverviewLogs } from "./api/keys/query-overview-logs";
import { keyVerificationsTimeseries } from "./api/keys/query-overview-timeseries";
import { enableKey } from "./api/keys/toggle-key-enabled";
import { overviewApiSearch } from "./api/overview-api-search";
import { queryApisOverview } from "./api/overview/query-overview";
import { queryVerificationTimeseries } from "./api/overview/query-timeseries";
import { queryApiKeyDetails } from "./api/query-api-key-details";
import { setDefaultApiBytes } from "./api/setDefaultBytes";
import { setDefaultApiPrefix } from "./api/setDefaultPrefix";
import { updateAPIDeleteProtection } from "./api/updateDeleteProtection";
import { updateApiIpWhitelist } from "./api/updateIpWhitelist";
import { updateApiName } from "./api/updateName";
import { fetchAuditLog } from "./audit/fetch";
import { auditLogsSearch } from "./audit/llm-search";
import { deletePermissionWithRelations } from "./authorization/permissions/delete";
import { permissionsLlmSearch } from "./authorization/permissions/llm-search";
import { queryPermissions } from "./authorization/permissions/query";
import { upsertPermission } from "./authorization/permissions/upsert";
import { getConnectedKeysAndPerms } from "./authorization/roles/connected-keys-and-perms";
import { deleteRoleWithRelations } from "./authorization/roles/delete";
import { queryRoleKeys } from "./authorization/roles/keys/connected-keys";
import { queryKeys } from "./authorization/roles/keys/query-keys";
import { searchKeys } from "./authorization/roles/keys/search-key";
import { rolesLlmSearch } from "./authorization/roles/llm-search";
import { queryRolePermissions } from "./authorization/roles/permissions/connected-permissions";
import { queryRolesPermissions } from "./authorization/roles/permissions/query-permissions";
import { searchRolesPermissions } from "./authorization/roles/permissions/search-permissions";
import { queryRoles } from "./authorization/roles/query";
import { upsertRole } from "./authorization/roles/upsert";
import { queryUsage } from "./billing/query-usage";
import { createProject } from "./deploy/project/create";
import { queryProjects } from "./deploy/project/list";
import { deploymentRouter } from "./deployment";
import { createIdentity } from "./identity/create";
import { queryIdentities } from "./identity/query";
import { searchIdentities } from "./identity/search";
import { createKey } from "./key/create";
import { createRootKey } from "./key/createRootKey";
import { deleteKeys } from "./key/delete";
import { fetchKeyPermissions } from "./key/fetch-key-permissions";
import { queryKeyDetailsLogs } from "./key/query-logs";
import { keyDetailsVerificationsTimeseries } from "./key/query-timeseries";
import { getConnectedRolesAndPerms } from "./key/rbac/connected-roles-and-perms";
import { getPermissionSlugs } from "./key/rbac/get-permission-slugs";
import { queryKeysPermissions } from "./key/rbac/permissions/query";
import { queryKeysRoles } from "./key/rbac/roles/query-keys-roles";
import { searchKeysRoles } from "./key/rbac/roles/search-keys-roles";
import { updateKeyRbac } from "./key/rbac/update-rbac";
import { updateKeysEnabled } from "./key/updateEnabled";
import { updateKeyExpiration } from "./key/updateExpiration";
import { updateKeyMetadata } from "./key/updateMetadata";
import { updateKeyName } from "./key/updateName";
import { updateKeyOwner } from "./key/updateOwnerId";
import { updateKeyRatelimit } from "./key/updateRatelimit";
import { updateKeyRemaining } from "./key/updateRemaining";
import { updateRootKeyName } from "./key/updateRootKeyName";
import { updateRootKeyPermissions } from "./key/updateRootKeyPermissions";
import { llmSearch } from "./logs/llm-search";
import { queryLogs } from "./logs/query-logs";
import { queryTimeseries } from "./logs/query-timeseries";
import {
  getInvitationList,
  getOrg,
  getOrganizationMemberList,
  inviteMember,
  removeMembership,
  revokeInvitation,
  updateMembership,
} from "./org";
import { createPlainIssue } from "./plain";
import { createNamespace } from "./ratelimit/createNamespace";
import { createOverride } from "./ratelimit/createOverride";
import { deleteNamespace } from "./ratelimit/deleteNamespace";
import { deleteOverride } from "./ratelimit/deleteOverride";
import { ratelimitLlmSearch } from "./ratelimit/llm-search";
import { searchNamespace } from "./ratelimit/namespace-search";
import { queryRatelimitLastUsed } from "./ratelimit/query-last-used-times";
import { queryRatelimitLatencyTimeseries } from "./ratelimit/query-latency-timeseries";
import { queryRatelimitLogs } from "./ratelimit/query-logs";
import { queryRatelimitWorkspaceDetails } from "./ratelimit/query-namespace-details";
import { queryRatelimitNamespaces } from "./ratelimit/query-namespaces";
import { queryRatelimitOverviewLogs } from "./ratelimit/query-overview-logs";
import { queryRatelimitTimeseries } from "./ratelimit/query-timeseries";
import { updateNamespaceName } from "./ratelimit/updateNamespaceName";
import { updateOverride } from "./ratelimit/updateOverride";
import { connectPermissionToRole } from "./rbac/connectPermissionToRole";
import { connectRoleToKey } from "./rbac/connectRoleToKey";
import { createPermission } from "./rbac/createPermission";
import { createRole } from "./rbac/createRole";
import { deletePermission } from "./rbac/deletePermission";
import { deleteRole } from "./rbac/deleteRole";
import { disconnectPermissionFromRole } from "./rbac/disconnectPermissionFromRole";
import { disconnectRoleFromKey } from "./rbac/disconnectRoleFromKey";
import { updatePermission } from "./rbac/updatePermission";
import { updateRole } from "./rbac/updateRole";
import { deleteRootKeys } from "./settings/root-keys/delete";
import { rootKeysLlmSearch } from "./settings/root-keys/llm-search";
import { queryRootKeys } from "./settings/root-keys/query";
import { cancelSubscription } from "./stripe/cancelSubscription";
import { createSubscription } from "./stripe/createSubscription";
import { uncancelSubscription } from "./stripe/uncancelSubscription";
import { updateSubscription } from "./stripe/updateSubscription";
import { getCurrentUser, listMemberships, switchOrg } from "./user";
import { vercelRouter } from "./vercel";
import { changeWorkspaceName } from "./workspace/changeName";
import { createWorkspace } from "./workspace/create";
import { onboardingKeyCreation } from "./workspace/onboarding";
import { optWorkspaceIntoBeta } from "./workspace/optIntoBeta";

export const router = t.router({
  key: t.router({
    create: createKey,
    delete: deleteKeys,
    fetchPermissions: fetchKeyPermissions,
    logs: t.router({
      query: queryKeyDetailsLogs,
      timeseries: keyDetailsVerificationsTimeseries,
    }),
    update: t.router({
      enabled: updateKeysEnabled,
      expiration: updateKeyExpiration,
      metadata: updateKeyMetadata,
      name: updateKeyName,
      ownerId: updateKeyOwner,
      ratelimit: updateKeyRatelimit,
      remaining: updateKeyRemaining,
      rbac: t.router({
        update: updateKeyRbac,
        roles: t.router({
          search: searchKeysRoles,
          query: queryKeysRoles,
        }),
        permissions: t.router({
          search: searchRolesPermissions,
          query: queryKeysPermissions,
        }),
      }),
    }),
    queryPermissionSlugs: getPermissionSlugs,
    connectedRolesAndPerms: getConnectedRolesAndPerms,
  }),
  rootKey: t.router({
    create: createRootKey,
    update: t.router({
      name: updateRootKeyName,
      // NOTE: permissions replaces the full permission set for a root key.
      // Clients must send the authoritative list to avoid lost updates.
      permissions: updateRootKeyPermissions,
    }),
  }),
  settings: t.router({
    rootKeys: t.router({
      query: queryRootKeys,
      llmSearch: rootKeysLlmSearch,
      delete: deleteRootKeys,
    }),
  }),
  api: t.router({
    create: createApi,
    delete: deleteApi,
    updateName: updateApiName,
    setDefaultPrefix: setDefaultApiPrefix,
    setDefaultBytes: setDefaultApiBytes,
    updateIpWhitelist: updateApiIpWhitelist,
    updateDeleteProtection: updateAPIDeleteProtection,
    queryApiKeyDetails,
    keys: t.router({
      timeseries: keyVerificationsTimeseries,
      activeKeysTimeseries: activeKeysTimeseries,
      query: queryKeysOverviewLogs,
      llmSearch: keysLlmSearch,
      list: queryKeysList,
      listLlmSearch: apiKeysLlmSearch,
      enableKey: enableKey,
      usageTimeseries: keyUsageTimeseries,
      latestVerification: keyLastVerificationTime,
    }),
    overview: t.router({
      timeseries: queryVerificationTimeseries,
      query: queryApisOverview,
      search: overviewApiSearch,
    }),
  }),
  workspace: t.router({
    create: createWorkspace,
    updateName: changeWorkspaceName,
    optIntoBeta: optWorkspaceIntoBeta,
    onboarding: onboardingKeyCreation,
  }),
  stripe: t.router({
    createSubscription,
    updateSubscription,
    cancelSubscription,
    uncancelSubscription,
  }),
  vercel: vercelRouter,
  plain: t.router({
    createIssue: createPlainIssue,
  }),
  authorization: t.router({
    permissions: t.router({
      query: queryPermissions,
      upsert: upsertPermission,
      delete: deletePermissionWithRelations,
      llmSearch: permissionsLlmSearch,
    }),
    roles: t.router({
      query: queryRoles,
      keys: t.router({
        search: searchKeys,
        query: queryKeys,
      }),
      permissions: t.router({
        search: searchRolesPermissions,
        query: queryRolesPermissions,
      }),
      upsert: upsertRole,
      delete: deleteRoleWithRelations,
      llmSearch: rolesLlmSearch,
      connectedKeysAndPerms: getConnectedKeysAndPerms,
      connectedKeys: queryRoleKeys,
      connectedPerms: queryRolePermissions,
    }),
  }),
  rbac: t.router({
    connectPermissionToRole: connectPermissionToRole,
    connectRoleToKey: connectRoleToKey,
    createPermission: createPermission,
    createRole: createRole,
    deletePermission: deletePermission,
    deleteRole: deleteRole,
    disconnectPermissionFromRole: disconnectPermissionFromRole,
    disconnectRoleFromKey: disconnectRoleFromKey,
    updatePermission: updatePermission,
    updateRole: updateRole,
  }),
  ratelimit: t.router({
    logs: t.router({
      query: queryRatelimitLogs,
      ratelimitLlmSearch,
      queryRatelimitTimeseries,
    }),
    overview: t.router({
      logs: t.router({
        query: queryRatelimitOverviewLogs,
        queryRatelimitLatencyTimeseries,
      }),
    }),
    namespace: t.router({
      queryRatelimitLastUsed,
      query: queryRatelimitNamespaces,
      queryDetails: queryRatelimitWorkspaceDetails,
      search: searchNamespace,
      create: createNamespace,
      update: t.router({
        name: updateNamespaceName,
      }),
      delete: deleteNamespace,
    }),
    override: t.router({
      create: createOverride,
      update: updateOverride,
      delete: deleteOverride,
    }),
  }),
  logs: t.router({
    queryLogs,
    queryTimeseries,
    llmSearch,
  }),
  billing: t.router({
    queryUsage,
  }),
  audit: t.router({
    logs: fetchAuditLog,
    llmSearch: auditLogsSearch,
  }),
  user: t.router({
    getCurrentUser,
    listMemberships,
    switchOrg,
  }),
  org: t.router({
    getOrg,
    members: t.router({
      list: getOrganizationMemberList,
      remove: removeMembership,
      update: updateMembership,
    }),
    invitations: t.router({
      list: getInvitationList,
      create: inviteMember,
      remove: revokeInvitation,
    }),
  }),
  identity: t.router({
    create: createIdentity,
    query: queryIdentities,
    search: searchIdentities,
  }),
  deploy: t.router({
    project: t.router({
      list: queryProjects,
      create: createProject,
    }),
  }),
  deployment: deploymentRouter,
});

// export type definition of API
export type Router = typeof router;
