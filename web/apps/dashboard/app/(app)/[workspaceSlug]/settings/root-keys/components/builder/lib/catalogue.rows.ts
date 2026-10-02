import { type PermissionRow, permissionRow } from "./catalogue.types";

// Every scope shows the same resources; only the path prefix above them moves.
// A policy on one app and a policy on every app in a project name the same
// rows, so the rows are built from the path of the thing that holds them.

// Logs are a resource of their own — a "/logs" leaf that only reads — so that a
// key can watch a keyspace without reading the keys in it.
function logRow(id: string, label: string, path: string): PermissionRow {
  return permissionRow({
    id,
    label,
    path,
    actions: { write: [], delete: [] },
  });
}

export function projectRow(projectPath: string): PermissionRow {
  return permissionRow({
    id: "project",
    label: "Projects",
    path: projectPath,
  });
}

export function appRow(appPath: string): PermissionRow {
  return permissionRow({ id: "app", label: "Apps", path: appPath });
}

export function environmentRows(environmentPath: string): PermissionRow[] {
  return [
    permissionRow({
      id: "environment",
      label: "Environments",
      path: environmentPath,
    }),
    permissionRow({
      id: "variable",
      label: "Environment variables",
      path: `${environmentPath}/variables/*`,
    }),
    permissionRow({
      id: "domain",
      label: "Domains",
      path: `${environmentPath}/domains/*`,
    }),
  ];
}

export function deploymentRows(environmentPath: string): PermissionRow[] {
  const deploymentPath = `${environmentPath}/deployments/*`;
  return [
    permissionRow({
      id: "deployment",
      label: "Deployments",
      path: deploymentPath,
    }),
    logRow("deployment_log", "Runtime logs", `${deploymentPath}/logs`),
    logRow("deployment_build_log", "Build logs", `${deploymentPath}/buildLogs`),
  ];
}

export function gatewayRows(environmentPath: string): PermissionRow[] {
  return [
    logRow("gateway_log", "HTTP request logs", `${environmentPath}/gateway/logs`),
    permissionRow({
      id: "gateway_policy",
      label: "Gateway policies",
      path: `${environmentPath}/gateway/policies/*`,
    }),
  ];
}

export function keyspaceRows(keyspacePath: string): PermissionRow[] {
  return [
    permissionRow({
      id: "keyspace",
      label: "Keyspaces",
      path: keyspacePath,
    }),
    logRow("keyspace_log", "Logs", `${keyspacePath}/logs`),
    permissionRow({
      id: "key",
      label: "Keys",
      path: `${keyspacePath}/keys/*`,
      actions: {
        verify: [{ name: "verify" }],
        decrypt: [{ name: "decrypt" }],
      },
    }),
  ];
}

export function namespaceRows(namespacePath: string): PermissionRow[] {
  return [
    permissionRow({
      id: "ratelimit_namespace",
      label: "Rate limit namespaces",
      path: namespacePath,
      actions: { limit: [{ name: "limit" }] },
    }),
    logRow("ratelimit_log", "Logs", `${namespacePath}/logs`),
    permissionRow({
      id: "ratelimit_override",
      label: "Rate limit overrides",
      path: `${namespacePath}/overrides/*`,
    }),
  ];
}

export function identityRows(projectPath: string): PermissionRow[] {
  return [
    permissionRow({
      id: "identity",
      label: "Identities",
      path: `${projectPath}/identities/*`,
    }),
  ];
}

export function rbacRows(projectPath: string): PermissionRow[] {
  return [
    permissionRow({
      id: "role",
      label: "Roles",
      path: `${projectPath}/rbac/roles/*`,
    }),
    permissionRow({
      id: "permission",
      label: "Permissions",
      path: `${projectPath}/rbac/permissions/*`,
    }),
  ];
}

export function portalRows(projectPath: string): PermissionRow[] {
  const portalPath = `${projectPath}/portals/*`;
  return [
    permissionRow({ id: "portal", label: "Portals", path: portalPath }),
    permissionRow({
      id: "portal_session",
      label: "Portal sessions",
      path: `${portalPath}/sessions/*`,
      actions: { delete: [] },
    }),
  ];
}

export function rootKeyRows(): PermissionRow[] {
  return [permissionRow({ id: "root_key", label: "Root keys", path: "rootKeys/*" })];
}

export function githubRows(): PermissionRow[] {
  return [
    permissionRow({
      id: "github_app",
      label: "GitHub apps",
      path: "github/apps/*",
    }),
  ];
}

export function limitsRows(): PermissionRow[] {
  return [
    permissionRow({
      id: "limits",
      label: "Limits",
      path: "limits",
      actions: { write: [], delete: [] },
    }),
  ];
}
