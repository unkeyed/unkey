import { t } from "../trpc";
import { createApi } from "./api/create";
import { deleteApi } from "./api/delete";
import { keysLlmSearch } from "./api/keys/llm-search";
import { activeKeysTimeseries } from "./api/keys/query-active-keys-timeseries";
import { queryKeysOverviewLogs } from "./api/keys/query-overview-logs";
import { keyVerificationsTimeseries } from "./api/keys/query-overview-timeseries";
import { overviewApiSearch } from "./api/overview-api-search";
import { queryApisOverview } from "./api/overview/query-overview";
import { queryVerificationTimeseries } from "./api/overview/query-timeseries";
import { setDefaultApiBytes } from "./api/setDefaultBytes";
import { setDefaultApiPrefix } from "./api/setDefaultPrefix";
import { updateAPIDeleteProtection } from "./api/updateDeleteProtection";
import { updateApiIpWhitelist } from "./api/updateIpWhitelist";
import { updateApiName } from "./api/updateName";
import { fetchAuditLog } from "./audit/fetch";
import { auditLogsSearch } from "./audit/llm-search";
import { createKey } from "./key/create";
import { createRootKey } from "./key/createRootKey";
import { deleteKeys } from "./key/delete";
import { deleteRootKeys } from "./key/deleteRootKey";
import { updateKeyEnabled } from "./key/updateEnabled";
import { updateKeyExpiration } from "./key/updateExpiration";
import { updateKeyMetadata } from "./key/updateMetadata";
import { updateKeyName } from "./key/updateName";
import { updateKeyOwnerId } from "./key/updateOwnerId";
import { updateKeyRatelimit } from "./key/updateRatelimit";
import { updateKeyRemaining } from "./key/updateRemaining";
import { updateRootKeyName } from "./key/updateRootKeyName";
import { llmSearch } from "./logs/llm-search";
import { queryLogs } from "./logs/query-logs";
import { queryTimeseries } from "./logs/query-timeseries";
import { createPlainIssue } from "./plain";
import { createNamespace } from "./ratelimit/createNamespace";
import { createOverride } from "./ratelimit/createOverride";
import { deleteNamespace } from "./ratelimit/deleteNamespace";
import { deleteOverride } from "./ratelimit/deleteOverride";
import { ratelimitLlmSearch } from "./ratelimit/llm-search";
import { searchNamespace } from "./ratelimit/namespace-search";
import { queryRatelimitLatencyTimeseries } from "./ratelimit/query-latency-timeseries";
import { queryRatelimitLogs } from "./ratelimit/query-logs";
import { queryRatelimitOverviewLogs } from "./ratelimit/query-overview-logs";
import { queryRatelimitTimeseries } from "./ratelimit/query-timeseries";
import { updateNamespaceName } from "./ratelimit/updateNamespaceName";
import { updateOverride } from "./ratelimit/updateOverride";
import { addPermissionToRootKey } from "./rbac/addPermissionToRootKey";
import { connectPermissionToRole } from "./rbac/connectPermissionToRole";
import { connectRoleToKey } from "./rbac/connectRoleToKey";
import { createPermission } from "./rbac/createPermission";
import { createRole } from "./rbac/createRole";
import { deletePermission } from "./rbac/deletePermission";
import { deleteRole } from "./rbac/deleteRole";
import { disconnectPermissionFromRole } from "./rbac/disconnectPermissionFromRole";
import { disconnectRoleFromKey } from "./rbac/disconnectRoleFromKey";
import { removePermissionFromRootKey } from "./rbac/removePermissionFromRootKey";
import { updatePermission } from "./rbac/updatePermission";
import { updateRole } from "./rbac/updateRole";
import { cancelSubscription } from "./stripe/cancelSubscription";
import { createSubscription } from "./stripe/createSubscription";
import { uncancelSubscription } from "./stripe/uncancelSubscription";
import { updateSubscription } from "./stripe/updateSubscription";
import { vercelRouter } from "./vercel";
import { changeWorkspaceName } from "./workspace/changeName";
import { createWorkspace } from "./workspace/create";
import { optWorkspaceIntoBeta } from "./workspace/optIntoBeta";

export const router = t.router({
  key: t.router({
    create: createKey,
    delete: deleteKeys,
    update: t.router({
      enabled: updateKeyEnabled,
      expiration: updateKeyExpiration,
      metadata: updateKeyMetadata,
      name: updateKeyName,
      ownerId: updateKeyOwnerId,
      ratelimit: updateKeyRatelimit,
      remaining: updateKeyRemaining,
    }),
  }),
  rootKey: t.router({
    create: createRootKey,
    delete: deleteRootKeys,
    update: t.router({
      name: updateRootKeyName,
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
    keys: t.router({
      timeseries: keyVerificationsTimeseries,
      activeKeysTimeseries: activeKeysTimeseries,
      query: queryKeysOverviewLogs,
      llmSearch: keysLlmSearch,
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
  rbac: t.router({
    addPermissionToRootKey: addPermissionToRootKey,
    connectPermissionToRole: connectPermissionToRole,
    connectRoleToKey: connectRoleToKey,
    createPermission: createPermission,
    createRole: createRole,
    deletePermission: deletePermission,
    deleteRole: deleteRole,
    disconnectPermissionFromRole: disconnectPermissionFromRole,
    disconnectRoleFromKey: disconnectRoleFromKey,
    removePermissionFromRootKey: removePermissionFromRootKey,
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
  audit: t.router({
    logs: fetchAuditLog,
    llmSearch: auditLogsSearch,
  }),
});

// export type definition of API
export type Router = typeof router;
