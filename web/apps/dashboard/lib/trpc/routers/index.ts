import { t } from "../trpc";
import { alerts } from "./alerts";
import { createApi } from "./api/create";
import { deleteApi } from "./api/delete";
import { keysLlmSearch } from "./api/keys/llm-search";
import { apiKeysLlmSearch } from "./api/keys/llm-search-api-keys";
import { activeKeysTimeseries } from "./api/keys/query-active-keys-timeseries";
import { queryKeysList } from "./api/keys/query-api-keys";
import { keyUsageTimeseries } from "./api/keys/query-key-usage-timeseries";
import { queryKeysOverviewLogs } from "./api/keys/query-overview-logs";
import { keyVerificationsTimeseries } from "./api/keys/query-overview-timeseries";
import { overviewApiSearch } from "./api/overview-api-search";
import { queryApisOverview } from "./api/overview/query-overview";
import { queryVerificationTimeseries } from "./api/overview/query-timeseries";
import { queryApiKeyDetails } from "./api/query-api-key-details";
import { setDefaultApiBytes } from "./api/setDefaultBytes";
import { setDefaultApiPrefix } from "./api/setDefaultPrefix";
import { updateAPIDeleteProtection } from "./api/updateDeleteProtection";
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
import { getDeployBudget, setDeployBudget } from "./billing/deploy-budget";
import { queryComputeAllocation } from "./billing/query-compute-allocation";
import { queryDeployUsage } from "./billing/query-deploy-usage";
import { queryDeployUsageBreakdown } from "./billing/query-deploy-usage-breakdown";
import { queryUsage } from "./billing/query-usage";
import { listApps } from "./deploy/app/list";
import { countCustomDomains } from "./deploy/custom-domains/count";
import { listDomainConnectHints } from "./deploy/custom-domains/hints";
import { authorizeDeployment } from "./deploy/deployment/authorize";
import { getDeploymentBuildSteps } from "./deploy/deployment/build-steps";
import { cancelDeployment } from "./deploy/deployment/cancel";
import { getDeploymentSteps } from "./deploy/deployment/deployment-steps";
import { getById as getDeploymentById } from "./deploy/deployment/getById";
import { getOpenApiDiff } from "./deploy/deployment/getOpenApiDiff";
import { getDeploymentInstanceEvents } from "./deploy/deployment/instance-events";
import { listDeployments } from "./deploy/deployment/list";
import { searchDeployments } from "./deploy/deployment/llm-search";
import { getDeploymentRuntimeLogs } from "./deploy/deployment/runtime-logs";
import { listDomains } from "./deploy/domains/list";
import { makeSensitive } from "./deploy/env-vars/make-sensitive";
import { renameEnvVars } from "./deploy/env-vars/rename";
import { getAvailableKeyspaces } from "./deploy/environment-settings/get-available-keyspaces";
import { getAvailableRegions } from "./deploy/environment-settings/get-available-regions";
import { generateRegex } from "./deploy/environment-settings/policies/generate-regex";
import { getAppRpsMetrics } from "./deploy/metrics/get-app-rps-metrics";
import { getDeploymentCpuTimeseries } from "./deploy/metrics/get-deployment-cpu-timeseries";
import { getDeploymentDiskTimeseries } from "./deploy/metrics/get-deployment-disk-timeseries";
import { getDeploymentInstanceCountTimeseries } from "./deploy/metrics/get-deployment-instance-count-timeseries";
import { getDeploymentLatencyMetrics } from "./deploy/metrics/get-deployment-latency-metrics";
import { getDeploymentMemoryTimeseries } from "./deploy/metrics/get-deployment-memory-timeseries";
import { getDeploymentNetworkEgressTimeseries } from "./deploy/metrics/get-deployment-network-egress-timeseries";
import { getDeploymentNetworkIngressTimeseries } from "./deploy/metrics/get-deployment-network-ingress-timeseries";
import { getDeploymentResourceSummary } from "./deploy/metrics/get-deployment-resource-summary";
import { getDeploymentRpsMetrics } from "./deploy/metrics/get-deployment-rps-metrics";
import { generateDeploymentTree } from "./deploy/network/generate";
import { getDeploymentTree } from "./deploy/network/get";
import { getInstanceRps } from "./deploy/network/get-instance-rps";
import { getRegionRps } from "./deploy/network/get-region-rps";
import { creationContext } from "./deploy/project/creation-context";
import { listProjects } from "./deploy/project/list";
import { createSharedSecret } from "./share/create";
import { revealSharedSecret } from "./share/reveal";

import { llmSearch as requestLogsLlmSearch } from "./deploy/request-logs/llm-search";
import { queryRequestLogs } from "./deploy/request-logs/query";
import { listInstances } from "./deploy/runtime-logs/list-instances";
import { llmSearch as runtimeLogsLlmSearch } from "./deploy/runtime-logs/llm-search";
import { queryRuntimeLogs } from "./deploy/runtime-logs/query";
import { listEnvironments } from "./environment/list";
import { listAllEnvironments } from "./environment/list-all";
import { githubRouter } from "./github";
import { queryIdentityLogs } from "./identity/query-logs";
import { queryIdentityTimeseries } from "./identity/query-timeseries";
import { createRootKey } from "./key/createRootKey";
import { fetchKeyPermissions } from "./key/fetch-key-permissions";
import { queryKeyDetailsLogs } from "./key/query-logs";
import { keyDetailsVerificationsTimeseries } from "./key/query-timeseries";
import { getConnectedRolesAndPerms } from "./key/rbac/connected-roles-and-perms";
import { getPermissionSlugs } from "./key/rbac/get-permission-slugs";
import { queryKeysPermissions } from "./key/rbac/permissions/query";
import { queryKeysRoles } from "./key/rbac/roles/query-keys-roles";
import { searchKeysRoles } from "./key/rbac/roles/search-keys-roles";
import { rerollRootKey } from "./key/reroll";
import { updateRootKeyName } from "./key/updateRootKeyName";
import { updateRootKeyPermissions } from "./key/updateRootKeyPermissions";
import { logdrain } from "./logdrain";
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
import { queryRatelimitLogEnrichment } from "./ratelimit/query-logs/enrichment";
import { queryRatelimitOverviewLogs } from "./ratelimit/query-overview-logs";
import { queryRatelimitTimeseries } from "./ratelimit/query-timeseries";
import { queryRatelimitTimeseriesBatch } from "./ratelimit/query-timeseries-batch";
import { updateNamespaceName } from "./ratelimit/updateNamespaceName";
import { updateOverride } from "./ratelimit/updateOverride";
import { connectPermissionToRole } from "./rbac/connectPermissionToRole";
import { connectRoleToKey } from "./rbac/connectRoleToKey";
import { createPermission } from "./rbac/createPermission";
import { createRole } from "./rbac/createRole";
import { deletePermission } from "./rbac/deletePermission";
import { disconnectPermissionFromRole } from "./rbac/disconnectPermissionFromRole";
import { disconnectRoleFromKey } from "./rbac/disconnectRoleFromKey";
import { updatePermission } from "./rbac/updatePermission";
import { updateRole } from "./rbac/updateRole";
import { deleteRootKeys } from "./settings/root-keys/delete";
import { rootKeysLlmSearch } from "./settings/root-keys/llm-search";
import { queryRootKeys } from "./settings/root-keys/query";
import { cancelDeploy } from "./stripe/cancelDeploy";
import { cancelSubscription } from "./stripe/cancelSubscription";
import { changeDeployPlan } from "./stripe/changeDeployPlan";
import { createSubscription } from "./stripe/createSubscription";
import { getBillingInfo } from "./stripe/getBillingInfo";
import { getCheckoutSession } from "./stripe/getCheckoutSession";
import { getDeployCredit } from "./stripe/getDeployCredit";
import { getDeployEntitlement } from "./stripe/getDeployEntitlement";
import { getDeployPlans } from "./stripe/getDeployPlans";
import { getDeploySubscription } from "./stripe/getDeploySubscription";
import { getProducts } from "./stripe/getProducts";
import { getSetupIntent } from "./stripe/getSetupIntent";
import { getSubscriptionPaymentUrl } from "./stripe/getSubscriptionPaymentUrl";
import { getUpcomingInvoice } from "./stripe/getUpcomingInvoice";
import { linkApiSubscription, linkDeploySubscription } from "./stripe/linkDeploySubscription";
import { seedTestCustomer } from "./stripe/seedTestCustomer";
import { subscribeDeploy } from "./stripe/subscribeDeploy";
import { uncancelSubscription } from "./stripe/uncancelSubscription";
import { updateCustomer } from "./stripe/updateCustomer";
import { updateSubscription } from "./stripe/updateSubscription";
import { updateWorkspaceStripeCustomer } from "./stripe/updateWorkspace";
import {
  getCurrentUser,
  listMemberships,
  listMfaFactors,
  removeMfaFactor,
  startMfaEnrollment,
  switchOrg,
  verifyMfaEnrollment,
} from "./user";
import { changeWorkspaceName } from "./workspace/changeName";
import { createWorkspace } from "./workspace/create";
import { getWorkspaceById } from "./workspace/getById";
import { getCurrentWorkspace } from "./workspace/getCurrent";
import { onboardingKeyCreation } from "./workspace/onboarding";

export const router = t.router({
  alerts,
  logdrain,
  share: t.router({
    create: createSharedSecret,
    reveal: revealSharedSecret,
  }),
  key: t.router({
    fetchPermissions: fetchKeyPermissions,
    logs: t.router({
      query: queryKeyDetailsLogs,
      timeseries: keyDetailsVerificationsTimeseries,
    }),
    update: t.router({
      rbac: t.router({
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
    reroll: rerollRootKey,
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
    updateDeleteProtection: updateAPIDeleteProtection,
    queryApiKeyDetails,
    keys: t.router({
      timeseries: keyVerificationsTimeseries,
      activeKeysTimeseries: activeKeysTimeseries,
      query: queryKeysOverviewLogs,
      llmSearch: keysLlmSearch,
      list: queryKeysList,
      listLlmSearch: apiKeysLlmSearch,
      usageTimeseries: keyUsageTimeseries,
    }),
    overview: t.router({
      timeseries: queryVerificationTimeseries,
      query: queryApisOverview,
      search: overviewApiSearch,
    }),
  }),
  workspace: t.router({
    create: createWorkspace,
    getCurrent: getCurrentWorkspace,
    getById: getWorkspaceById,
    updateName: changeWorkspaceName,
    onboarding: onboardingKeyCreation,
  }),
  stripe: t.router({
    createSubscription,
    updateSubscription,
    cancelSubscription,
    uncancelSubscription,
    getBillingInfo,
    updateCustomer,
    getCheckoutSession,
    getProducts,
    getSetupIntent,
    getSubscriptionPaymentUrl,
    updateWorkspaceStripeCustomer,
    subscribeDeploy,
    linkApiSubscription,
    linkDeploySubscription,
    changeDeployPlan,
    cancelDeploy,
    seedTestCustomer,
    getDeploySubscription,
    getDeployPlans,
    getDeployCredit,
    getDeployEntitlement,
    getUpcomingInvoice,
  }),
  github: githubRouter,
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
    disconnectPermissionFromRole: disconnectPermissionFromRole,
    disconnectRoleFromKey: disconnectRoleFromKey,
    updatePermission: updatePermission,
    updateRole: updateRole,
  }),
  ratelimit: t.router({
    logs: t.router({
      query: queryRatelimitLogs,
      enrichment: queryRatelimitLogEnrichment,
      ratelimitLlmSearch,
      queryRatelimitTimeseries,
      queryRatelimitTimeseriesBatch,
    }),
    overview: t.router({
      logs: t.router({
        query: queryRatelimitOverviewLogs,
        queryRatelimitLatencyTimeseries,
      }),
    }),
    namespace: t.router({
      list: listRatelimitNamespaces,
      queryRatelimitLastUsed,
      create: createNamespace,
      update: t.router({
        name: updateNamespaceName,
      }),
      delete: deleteNamespace,
    }),
    override: t.router({
      list: listRatelimitOverrides,
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
    queryDeployUsage,
    queryDeployUsageBreakdown,
    queryComputeAllocation,
    getDeployBudget,
    setDeployBudget,
  }),
  audit: t.router({
    logs: fetchAuditLog,
    llmSearch: auditLogsSearch,
  }),
  user: t.router({
    getCurrentUser,
    listMemberships,
    switchOrg,
    mfa: t.router({
      listFactors: listMfaFactors,
      startEnrollment: startMfaEnrollment,
      verifyEnrollment: verifyMfaEnrollment,
      removeFactor: removeMfaFactor,
    }),
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
    logs: t.router({
      query: queryIdentityLogs,
      timeseries: queryIdentityTimeseries,
    }),
  }),
  deploy: t.router({
    network: t.router({
      generate: generateDeploymentTree,
      get: getDeploymentTree,
      getInstanceRps,
      getRegionRps,
    }),
    project: t.router({
      list: listProjects,
      creationContext,
    }),
    app: t.router({
      list: listApps,
    }),
    environmentSettings: t.router({
      getAvailableRegions,
      getAvailableKeyspaces,
      policies: t.router({
        generateRegex,
      }),
    }),
    environment: t.router({
      list: listEnvironments,
      listAll: listAllEnvironments,
    }),
    envVar: t.router({
      rename: renameEnvVars,
      makeSensitive,
    }),
    domain: t.router({
      list: listDomains,
    }),
    customDomain: t.router({
      count: countCustomDomains,
      hints: listDomainConnectHints,
    }),
    deployment: t.router({
      list: listDeployments,
      getById: getDeploymentById,
      buildSteps: getDeploymentBuildSteps,
      runtimeLogs: getDeploymentRuntimeLogs,
      instanceEvents: getDeploymentInstanceEvents,
      steps: getDeploymentSteps,
      search: searchDeployments,
      getOpenApiDiff: getOpenApiDiff,
      authorize: authorizeDeployment,
      cancel: cancelDeployment,
    }),
    requestLogs: t.router({
      query: queryRequestLogs,
      llmSearch: requestLogsLlmSearch,
    }),
    runtimeLogs: t.router({
      query: queryRuntimeLogs,
      llmSearch: runtimeLogsLlmSearch,
      listInstances,
    }),
    metrics: t.router({
      getAppRpsMetrics,
      getDeploymentRpsMetrics,
      getDeploymentLatencyMetrics,
      getDeploymentCpuTimeseries,
      getDeploymentMemoryTimeseries,
      getDeploymentDiskTimeseries,
      getDeploymentNetworkEgressTimeseries,
      getDeploymentNetworkIngressTimeseries,
      getDeploymentInstanceCountTimeseries,
      getDeploymentResourceSummary,
    }),
  }),
});

// export type definition of API
export type Router = typeof router;
