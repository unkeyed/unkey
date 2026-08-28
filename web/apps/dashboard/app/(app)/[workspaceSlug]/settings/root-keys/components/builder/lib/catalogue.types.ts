// The coarse actions below are a UI vocabulary. Each one expands into the
// resource-suffixed action names of the permission catalog in `@unkey/rbac`
// ("write" on the key row is `write_key`), which is what the URN carries.
export const ACTIONS = ["read", "write", "delete", "verify", "decrypt", "limit"] as const;

export type Action = (typeof ACTIONS)[number];

export const CRUD_ACTIONS: readonly Action[] = ["read", "write", "delete"];

export const READ_ACTIONS: readonly Action[] = ["read"];

export const READ_WRITE_ACTIONS: readonly Action[] = ["read", "write"];

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
  return path.split(INSTANCE_TOKEN).join(instance);
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

// Create and update are the same privilege, so a row's mutation is a single
// `write_<resource>`. Verify, decrypt and limit are narrower than a read and
// belong only to the rows that declare them.
export function crud(resource: string): Record<Action, readonly GrantSpec[]> {
  return {
    read: [{ name: `read_${resource}` }],
    write: [{ name: `write_${resource}` }],
    delete: [{ name: `delete_${resource}` }],
    verify: [],
    decrypt: [],
    limit: [],
  };
}

export function permissionRow(spec: PermissionRowSpec): PermissionRow {
  const convention = crud(spec.resource);
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
  // What `{instance}` stands for when a policy covers every instance. An app
  // lives at `projects/{project}/apps/{app}`, so "all apps" is that whole path
  // wildcarded, not a lone "*".
  allInstance: string;
  groups: CatalogueGroup[];
};

export function resolveInstance(catalogue: ScopeCatalogue, instance: string): string {
  return instance === ALL_INSTANCES ? catalogue.allInstance : instance;
}

export type PermissionSelection = Record<string, Action[]>;
