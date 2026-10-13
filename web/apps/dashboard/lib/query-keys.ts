import type { UsagePeriod } from "@/app/(app)/[workspaceSlug]/settings/usage/period";
const keysRoot = ["keys"] as const;
const rbacRoot = ["rbac"] as const;
const rolesRoot = [...rbacRoot, "roles"] as const;
const permissionsRoot = [...rbacRoot, "permissions"] as const;
const identitiesRoot = ["identities"] as const;
const workspaceRoot = ["workspace"] as const;
const ratelimitNamespacesRoot = ["ratelimit", "namespaces"] as const;
const portalSessionLists = (portalId: string) => ["portalSessions", portalId, "list"] as const;

export const queryKeys = {
  apis: {
    detail: (apiId: string) => ["apis", "detail", apiId] as const,
  },
  deployments: {
    buildLogs: (deploymentId: string) => ["deployments", "buildLogs", deploymentId] as const,
  },
  identities: {
    all: identitiesRoot,
    workspace: (workspaceId: string) => [...identitiesRoot, workspaceId] as const,
    lists: (workspaceId: string) => [...identitiesRoot, workspaceId, "list"] as const,
    list: (workspaceId: string, search: string) =>
      [...identitiesRoot, workspaceId, "list", search] as const,
    details: (workspaceId: string) => [...identitiesRoot, workspaceId, "detail"] as const,
    detail: (workspaceId: string, identityId: string) =>
      [...identitiesRoot, workspaceId, "detail", identityId] as const,
  },
  keys: {
    all: keysRoot,
    detail: (keyId: string) => [...keysRoot, "detail", keyId] as const,
  },
  portal: {
    detail: (keyAuthId: string) => ["portal", keyAuthId] as const,
    sessionLists: portalSessionLists,
    sessions: (portalId: string, search: string) =>
      [...portalSessionLists(portalId), search] as const,
  },
  ratelimit: {
    namespaces: {
      all: ratelimitNamespacesRoot,
      list: (search: string) => [...ratelimitNamespacesRoot, "list", search] as const,
      detail: (namespaceId: string) => [...ratelimitNamespacesRoot, "detail", namespaceId] as const,
      every: [...ratelimitNamespacesRoot, "every"] as const,
    },
  },
  rbac: {
    all: rbacRoot,
    roles: {
      list: (limit: number) => [...rolesRoot, "list", limit] as const,
      search: (query: string) => [...rolesRoot, "search", query] as const,
      detail: (roleName: string) => [...rolesRoot, "detail", roleName] as const,
    },
    permissions: {
      list: (limit: number) => [...permissionsRoot, "list", limit] as const,
      search: (query: string) => [...permissionsRoot, "search", query] as const,
    },
  },
  statusPage: {
    summary: ["status-page-summary"] as const,
  },
  workspace: {
    all: workspaceRoot,
    limits: [...workspaceRoot, "limits"] as const,
    usage: (period: UsagePeriod) => [...workspaceRoot, "usage", period] as const,
  },
};
