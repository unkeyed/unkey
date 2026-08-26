import { type PermissionRow, permissionRow } from "./catalogue.types";

// Every scope shows the same resources; only the path prefix above them moves.
// A policy on one app and a policy on every app in a project name the same
// rows, so the rows are built from the path of the thing that holds them.

// Logs are a resource of their own — a "/logs" leaf that only reads — so that a
// key can watch a keyspace without reading the keys in it.
function logRow(id: string, label: string, path: string, action: string): PermissionRow {
  return permissionRow({
    id,
    label,
    path,
    resource: id,
    actions: { read: [{ name: action }], write: [], delete: [] },
  });
}

export function projectRow(projectPath: string): PermissionRow {
  return permissionRow({
    id: "project",
    label: "Projects",
    path: projectPath,
    resource: "project",
  });
}

export function appRow(appPath: string): PermissionRow {
  return permissionRow({ id: "app", label: "Apps", path: appPath, resource: "app" });
}

export function environmentRows(environmentPath: string): PermissionRow[] {
  return [
    permissionRow({
      id: "environment",
      label: "Environments",
      path: environmentPath,
      resource: "environment",
    }),
    permissionRow({
      id: "variable",
      label: "Environment variables",
      path: `${environmentPath}/variables/*`,
      resource: "environment_variable",
    }),
    permissionRow({
      id: "domain",
      label: "Domains",
      path: `${environmentPath}/domains/*`,
      resource: "domain",
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
      resource: "deployment",
    }),
    logRow("deployment_log", "Runtime logs", `${deploymentPath}/logs`, "read_deployment_logs"),
  ];
}

export function gatewayRows(environmentPath: string): PermissionRow[] {
  return [
    logRow(
      "gateway_log",
      "HTTP request logs",
      `${environmentPath}/gateway/logs`,
      "read_gateway_logs",
    ),
    permissionRow({
      id: "gateway_policy",
      label: "Gateway policies",
      path: `${environmentPath}/gateway/policies/*`,
      resource: "gateway_policy",
    }),
  ];
}

export function keyspaceRows(keyspacePath: string): PermissionRow[] {
  return [
    permissionRow({
      id: "keyspace",
      label: "Keyspaces",
      path: keyspacePath,
      resource: "keyspace",
    }),
    logRow("keyspace_log", "Logs", `${keyspacePath}/logs`, "read_keyspace_logs"),
    permissionRow({
      id: "key",
      label: "Keys",
      path: `${keyspacePath}/keys/*`,
      resource: "key",
      actions: {
        verify: [{ name: "verify_key" }],
        decrypt: [{ name: "decrypt_key" }],
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
      resource: "ratelimit_namespace",
      actions: { limit: [{ name: "limit_ratelimit_namespace" }] },
    }),
    logRow("ratelimit_log", "Logs", `${namespacePath}/logs`, "read_ratelimit_logs"),
    permissionRow({
      id: "ratelimit_override",
      label: "Rate limit overrides",
      path: `${namespacePath}/overrides/*`,
      resource: "ratelimit_override",
    }),
  ];
}

export function identityRows(projectPath: string): PermissionRow[] {
  return [
    permissionRow({
      id: "identity",
      label: "Identities",
      path: `${projectPath}/identities/*`,
      resource: "identity",
    }),
  ];
}

export function rbacRows(projectPath: string): PermissionRow[] {
  return [
    permissionRow({
      id: "role",
      label: "Roles",
      path: `${projectPath}/rbac/roles/*`,
      resource: "role",
    }),
    permissionRow({
      id: "permission",
      label: "Permissions",
      path: `${projectPath}/rbac/permissions/*`,
      resource: "permission",
    }),
  ];
}

export function githubRows(): PermissionRow[] {
  return [
    permissionRow({
      id: "github_app",
      label: "GitHub apps",
      path: "github/apps/*",
      resource: "github_app",
    }),
  ];
}
