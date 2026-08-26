// The coarse actions below are a UI vocabulary. The Go action vocabulary
// (`pkg/rbac/permissions/*.go`) is still partial, so some catalogue rows expand
// into URNs the API cannot enforce yet.
export const ACTIONS = ["read", "write", "delete", "decrypt"] as const;

export type Action = (typeof ACTIONS)[number];

export const COARSE_ACTIONS: readonly Action[] = ["read", "write", "delete"];

export const RESOURCE_SCOPES = [
  "workspace",
  "projects",
  "apps",
  "environments",
  "keyspaces",
  "ratelimit-namespaces",
  "identities",
  "rbac",
] as const;

export type ResourceScope = (typeof RESOURCE_SCOPES)[number];

export const INSTANCE_TOKEN = "{instance}";

export const ALL_INSTANCES = "__all__";

export function instancePath(path: string, instance: string): string {
  return path.split(INSTANCE_TOKEN).join(instance === ALL_INSTANCES ? "*" : instance);
}

export type ActionGrant = {
  name: string;
  path: string;
};

export type GrantSpec = {
  name: string;
  path?: string;
};

export type PermissionRow = {
  id: string;
  label: string;
  path: string;
  actions: Record<Action, readonly ActionGrant[]>;
};

type PermissionRowSpec = {
  id: string;
  label: string;
  path: string;
  resource: string;
  actions?: Partial<Record<Action, readonly GrantSpec[]>>;
};

export function permissionRow(spec: PermissionRowSpec): PermissionRow {
  const convention: Record<Action, readonly GrantSpec[]> = {
    read: [{ name: `read_${spec.resource}` }],
    write: [{ name: `create_${spec.resource}` }, { name: `update_${spec.resource}` }],
    delete: [{ name: `delete_${spec.resource}` }],
    decrypt: [],
  };
  const actions = {} as Record<Action, readonly ActionGrant[]>;
  for (const action of ACTIONS) {
    actions[action] = (spec.actions?.[action] ?? convention[action]).map((grant) => ({
      name: grant.name,
      path: grant.path ?? spec.path,
    }));
  }
  return { id: spec.id, label: spec.label, path: spec.path, actions };
}

export function rowOffers(row: PermissionRow, action: Action): boolean {
  return row.actions[action].length > 0;
}

export function offeredActions(row: PermissionRow): Action[] {
  return ACTIONS.filter((action) => rowOffers(row, action));
}

export type CatalogueGroup = {
  id: string;
  label: string;
  rows: PermissionRow[];
};

export type ScopeCatalogue = {
  scope: ResourceScope;
  label: string;
  allLabel: string;
  instanceNoun: string | null;
  groups: CatalogueGroup[];
};

export type PermissionSelection = Record<string, Action[]>;
