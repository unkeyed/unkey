import { router } from "../trpc";
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
import { getDeploymentBuildSteps } from "./deploy/deployment/build-steps";
import { getOpenApiDiff } from "./deploy/deployment/getOpenApiDiff";
import { listDeployments } from "./deploy/deployment/list";
import { searchDeployments } from "./deploy/deployment/llm-search";
import { promote } from "./deploy/deployment/promote";
import { rollback } from "./deploy/deployment/rollback";
import { listDomains } from "./deploy/domains/list";
import { getEnvs } from "./deploy/envs/list";
import { createProject } from "./deploy/project/create";
import { listProjects } from "./deploy/project/list";
import { listEnvironments } from "./environment/list";
import { createIdentity } from "./identity/create";
import { getIdentityById } from "./identity/getById";
import { queryIdentities } from "./identity/query";
import { searchIdentities } from "./identity/search";
import { searchIdentitiesWithRelations } from "./identity/searchWithRelations";
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
import { listRatelimitNamespaces } from "./ratelimit/namespaces_list";
import { listRatelimitOverrides } from "./ratelimit/overrides_list";
import { queryRatelimitLastUsed } from "./ratelimit/query-last-used-times";
import { queryRatelimitLatencyTimeseries } from "./ratelimit/query-latency-timeseries";
import { queryRatelimitLogs } from "./ratelimit/query-logs";
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
import { getBillingInfo } from "./stripe/getBillingInfo";
import { getCheckoutSession } from "./stripe/getCheckoutSession";
import { getCustomer } from "./stripe/getCustomer";
import { getProducts } from "./stripe/getProducts";
import { getSetupIntent } from "./stripe/getSetupIntent";
import { uncancelSubscription } from "./stripe/uncancelSubscription";
import { updateCustomer } from "./stripe/updateCustomer";
import { updateSubscription } from "./stripe/updateSubscription";
import { updateWorkspaceStripeCustomer } from "./stripe/updateWorkspace";
import { getCurrentUser, listMemberships, switchOrg } from "./user";
import { vercelRouter } from "./vercel";
import { changeWorkspaceName } from "./workspace/changeName";
import { createWorkspace } from "./workspace/create";
import { getWorkspaceById } from "./workspace/getById";
import { getCurrentWorkspace } from "./workspace/getCurrent";
import { onboardingKeyCreation } from "./workspace/onboarding";
import { optWorkspaceIntoBeta } from "./workspace/optIntoBeta";

export const appRouter = router({
  key: router({
    create: createKey,
    delete: deleteKeys,
    fetchPermissions: fetchKeyPermissions,
    logs: router({
      query: queryKeyDetailsLogs,
      timeseries: keyDetailsVerificationsTimeseries,
    }),
    update: router({
      enabled: updateKeysEnabled,
      expiration: updateKeyExpiration,
      metadata: updateKeyMetadata,
      name: updateKeyName,
      ownerId: updateKeyOwner,
      ratelimit: updateKeyRatelimit,
      remaining: updateKeyRemaining,
      rbac: router({
        update: updateKeyRbac,
        roles: router({
          search: searchKeysRoles,
          query: queryKeysRoles,
        }),
        permissions: router({
          search: searchRolesPermissions,
          query: queryKeysPermissions,
        }),
      }),
    }),
    queryPermissionSlugs: getPermissionSlugs,
    connectedRolesAndPerms: getConnectedRolesAndPerms,
  }),
  rootKey: router({
    create: createRootKey,
    update: router({
      name: updateRootKeyName,
      // NOTE: permissions replaces the full permission set for a root key.
      // Clients must send the authoritative list to avoid lost updates.
      permissions: updateRootKeyPermissions,
    }),
  }),
  settings: router({
    rootKeys: router({
      query: queryRootKeys,
      llmSearch: rootKeysLlmSearch,
      delete: deleteRootKeys,
    }),
  }),
  api: router({
    create: createApi,
    delete: deleteApi,
    updateName: updateApiName,
    setDefaultPrefix: setDefaultApiPrefix,
    setDefaultBytes: setDefaultApiBytes,
    updateIpWhitelist: updateApiIpWhitelist,
    updateDeleteProtection: updateAPIDeleteProtection,
    queryApiKeyDetails,
    keys: router({
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
    overview: router({
      timeseries: queryVerificationTimeseries,
      query: queryApisOverview,
      search: overviewApiSearch,
    }),
  }),
  workspace: router({
    create: createWorkspace,
    getCurrent: getCurrentWorkspace,
    getById: getWorkspaceById,
    updateName: changeWorkspaceName,
    optIntoBeta: optWorkspaceIntoBeta,
    onboarding: onboardingKeyCreation,
  }),
  stripe: router({
    createSubscription,
    updateSubscription,
    cancelSubscription,
    uncancelSubscription,
    getBillingInfo,
    updateCustomer,
    getCheckoutSession,
    getCustomer,
    getProducts,
    getSetupIntent,
    updateWorkspaceStripeCustomer,
  }),
  vercel: vercelRouter,
  plain: router({
    createIssue: createPlainIssue,
  }),
  authorization: router({
    permissions: router({
      query: queryPermissions,
      upsert: upsertPermission,
      delete: deletePermissionWithRelations,
      llmSearch: permissionsLlmSearch,
    }),
    roles: router({
      query: queryRoles,
      keys: router({
        search: searchKeys,
        query: queryKeys,
      }),
      permissions: router({
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
  rbac: router({
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
  ratelimit: router({
    logs: router({
      query: queryRatelimitLogs,
      ratelimitLlmSearch,
      queryRatelimitTimeseries,
    }),
    overview: router({
      logs: router({
        query: queryRatelimitOverviewLogs,
        queryRatelimitLatencyTimeseries,
      }),
    }),
    namespace: router({
      list: listRatelimitNamespaces,
      queryRatelimitLastUsed,
      create: createNamespace,
      update: router({
        name: updateNamespaceName,
      }),
      delete: deleteNamespace,
    }),
    override: router({
      list: listRatelimitOverrides,
      create: createOverride,
      update: updateOverride,
      delete: deleteOverride,
    }),
  }),
  logs: router({
    queryLogs,
    queryTimeseries,
    llmSearch,
  }),
  billing: router({
    queryUsage,
  }),
  audit: router({
    logs: fetchAuditLog,
    llmSearch: auditLogsSearch,
  }),
  user: router({
    getCurrentUser,
    listMemberships,
    switchOrg,
  }),
  org: router({
    getOrg,
    members: router({
      list: getOrganizationMemberList,
      remove: removeMembership,
      update: updateMembership,
    }),
    invitations: router({
      list: getInvitationList,
      create: inviteMember,
      remove: revokeInvitation,
    }),
  }),
  identity: router({
    searchWithRelations: searchIdentitiesWithRelations,
    create: createIdentity,
    query: queryIdentities,
    search: searchIdentities,
    getById: getIdentityById,
  }),
  deploy: router({
    project: router({
      list: listProjects,
      create: createProject,
    }),
    environment: router({
      list_dummy: getEnvs,
      list: listEnvironments,
    }),
    domain: router({
      list: listDomains,
    }),
    deployment: router({
      list: listDeployments,
      buildSteps: getDeploymentBuildSteps,
      search: searchDeployments,
      getOpenApiDiff: getOpenApiDiff,
      rollback,
      promote,
    }),
  }),
});

export type AppRouter = typeof appRouter;
