/**
 * Resource catalog for the permissions lab. Path shapes come from
 * docs/engineering/architecture/resources/unkey-resource-names.mdx; actions
 * come from pkg/rbac/permissions plus the create-on-parent examples in
 * docs/engineering/architecture/authorization/resource-permissions.mdx.
 *
 * Prototype-only: some actions are extrapolated CRUD verbs so screens have
 * enough material; the implemented Go set is marked with implemented: true.
 */

export interface ActionDef {
  action: string;
  description: string;
  /** true when the action exists in pkg/rbac/permissions today */
  implemented: boolean;
}

export interface ResourceTypeDef {
  type: string;
  label: string;
  /** Path shape segments; "{...}" marks an id slot. */
  shape: string[];
  actions: ActionDef[];
}

const a = (action: string, description: string, implemented = false): ActionDef => ({
  action,
  description,
  implemented,
});

export const RESOURCE_TYPES: ResourceTypeDef[] = [
  {
    type: "keyspace",
    label: "Keyspace",
    shape: ["keyspaces", "{keyspace_id}"],
    actions: [
      a("read_keyspace", "Read keyspace configuration", true),
      a("create_key", "Create keys in this keyspace", true),
      a("create_keyspace", "Create keyspaces (top-level, use keyspaces/*)"),
      a("update_keyspace", "Update keyspace configuration"),
      a("delete_keyspace", "Delete this keyspace"),
    ],
  },
  {
    type: "key",
    label: "Key",
    shape: ["keyspaces", "{keyspace_id}", "keys", "{key_id}"],
    actions: [
      a("read_key", "Read key metadata", true),
      a("update_key", "Update key metadata", true),
      a("verify_key", "Verify this key", true),
      a("encrypt_key", "Create recoverable encrypted keys", true),
      a("decrypt_key", "Decrypt recoverable key material", true),
      a("delete_key", "Delete this key", true),
    ],
  },
  {
    type: "identity",
    label: "Identity",
    shape: ["identities", "{identity_id}"],
    actions: [
      a("create_identity", "Create identities (top-level, use identities/*)", true),
      a("read_identity", "Read identity data", true),
      a("update_identity", "Update identity data", true),
      a("delete_identity", "Delete this identity", true),
    ],
  },
  {
    type: "ratelimit_namespace",
    label: "Ratelimit namespace",
    shape: ["ratelimits", "namespaces", "{namespace_id}"],
    actions: [
      a("read_namespace", "Read namespace configuration"),
      a("set_override", "Set overrides in this namespace", true),
      a("create_namespace", "Create namespaces (top-level, use ratelimits/namespaces/*)"),
      a("delete_namespace", "Delete this namespace"),
    ],
  },
  {
    type: "ratelimit_override",
    label: "Ratelimit override",
    shape: ["ratelimits", "namespaces", "{namespace_id}", "overrides", "{override_id}"],
    actions: [
      a("read_override", "Read this override", true),
      a("delete_override", "Delete this override", true),
    ],
  },
  {
    type: "role",
    label: "Role",
    shape: ["rbac", "roles", "{role_id}"],
    actions: [
      a("read_role", "Read this role"),
      a("update_role", "Update this role"),
      a("create_role", "Create roles (top-level, use rbac/roles/*)"),
      a("delete_role", "Delete this role"),
      a("add_role", "Attach this role to a principal"),
      a("remove_role", "Detach this role from a principal"),
    ],
  },
  {
    type: "permission",
    label: "Permission",
    shape: ["rbac", "permissions", "{permission_id}"],
    actions: [
      a("read_permission", "Read this permission"),
      a("create_permission", "Create permissions", true),
      a("delete_permission", "Delete this permission"),
      a("add_permission", "Attach this permission to a principal"),
      a("remove_permission", "Detach this permission from a principal"),
    ],
  },
  {
    type: "project",
    label: "Project",
    shape: ["projects", "{project_id}"],
    actions: [
      a("read_project", "Read project configuration"),
      a("create_project", "Create projects (top-level, use projects/*)"),
      a("update_project", "Update project configuration"),
      a("delete_project", "Delete this project"),
      a("create_app", "Create apps in this project", true),
    ],
  },
  {
    type: "app",
    label: "App",
    shape: ["projects", "{project_id}", "apps", "{app_id}"],
    actions: [
      a("read_app", "Read app configuration"),
      a("update_app", "Update app configuration"),
      a("delete_app", "Delete this app"),
      a("create_environment", "Create environments in this app", true),
    ],
  },
  {
    type: "environment",
    label: "Environment",
    shape: ["projects", "{project_id}", "apps", "{app_id}", "environments", "{environment_id}"],
    actions: [
      a("read_environment", "Read environment configuration", true),
      a("update_environment", "Update environment configuration", true),
      a("create_deployment", "Create deployments in this environment", true),
      a("create_domain", "Create domains in this environment", true),
      a("create_variable", "Create variables in this environment", true),
    ],
  },
  {
    type: "deployment",
    label: "Deployment",
    shape: [
      "projects",
      "{project_id}",
      "apps",
      "{app_id}",
      "environments",
      "{environment_id}",
      "deployments",
      "{deployment_id}",
    ],
    actions: [
      a("read_deployment", "Read this deployment"),
      a("delete_deployment", "Delete this deployment"),
    ],
  },
  {
    type: "domain",
    label: "Domain",
    shape: [
      "projects",
      "{project_id}",
      "apps",
      "{app_id}",
      "environments",
      "{environment_id}",
      "domains",
      "{domain_id}",
    ],
    actions: [a("read_domain", "Read this domain"), a("delete_domain", "Delete this domain")],
  },
  {
    type: "variable",
    label: "Variable",
    shape: [
      "projects",
      "{project_id}",
      "apps",
      "{app_id}",
      "environments",
      "{environment_id}",
      "variables",
      "{variable_id}",
    ],
    actions: [
      a("read_variable", "Read this variable"),
      a("update_variable", "Update this variable"),
      a("delete_variable", "Delete this variable"),
    ],
  },
];

export const GLOBAL_ACTION_WILDCARD = "*";

function isIdSlot(shapeSegment: string): boolean {
  return shapeSegment.startsWith("{");
}

/**
 * Matches a concrete or pattern path against the catalog shapes. A trailing
 * "**" matches every type whose shape extends the base path, so callers get
 * the union of possible target types.
 */
export function resourceTypesOfPath(path: string): ResourceTypeDef[] {
  const segments = path.split("/");
  const hasDescendants = segments[segments.length - 1] === "**";
  const base = hasDescendants ? segments.slice(0, -1) : segments;

  return RESOURCE_TYPES.filter((def) => {
    if (hasDescendants) {
      // "**" covers the base itself and everything below it.
      if (def.shape.length < base.length) {
        return false;
      }
      return shapeMatches(def.shape.slice(0, base.length), base);
    }
    return def.shape.length === base.length && shapeMatches(def.shape, base);
  });
}

function shapeMatches(shape: string[], segments: string[]): boolean {
  for (let i = 0; i < shape.length; i++) {
    if (isIdSlot(shape[i])) {
      // Any id or "*" fills an id slot; collection labels do not.
      if (segments[i] === "**") {
        return false;
      }
      continue;
    }
    if (shape[i] !== segments[i] && segments[i] !== "*") {
      return false;
    }
  }
  return true;
}

/**
 * Actions valid for a resource path or pattern. For a descendant scope the
 * result is the union across every covered type. For the global "**" the
 * wildcard action "*" is included (the admin escape hatch).
 */
export function actionsForPath(path: string): ActionDef[] {
  const types = resourceTypesOfPath(path);
  const seen = new Map<string, ActionDef>();
  for (const t of types) {
    for (const action of t.actions) {
      if (!seen.has(action.action)) {
        seen.set(action.action, action);
      }
    }
  }
  const actions = [...seen.values()].sort((x, y) => x.action.localeCompare(y.action));
  if (path === "**") {
    actions.unshift(a(GLOBAL_ACTION_WILDCARD, "Every action (admin escape hatch)", true));
  }
  return actions;
}

/** The single best-matching type for a concrete path, if any. */
export function resourceTypeOfPath(path: string): ResourceTypeDef | null {
  const matches = resourceTypesOfPath(path).filter(
    (def) => def.shape.length === path.split("/").length,
  );
  return matches[0] ?? null;
}
