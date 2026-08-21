// The three coarse actions below are a UI vocabulary. The Go action vocabulary
// (`pkg/rbac/permissions/*.go`) is still partial, so some catalogue rows expand
// into URNs the API cannot enforce yet.
export const ACTIONS = ["read", "write", "delete"] as const;

export type Action = (typeof ACTIONS)[number];

export const RESOURCE_SCOPES = [
  "workspace",
  "projects",
  "apps",
  "environments",
  "keyspaces",
  "ratelimit-namespaces",
] as const;

export type ResourceScope = (typeof RESOURCE_SCOPES)[number];

export const INSTANCE_TOKEN = "{instance}";

export type ActionGrant = {
  name: string;
  path?: string;
};

export type PermissionRow = {
  id: string;
  label: string;
  description: string;
  path: string;
  resource: string;
  actions?: Partial<Record<Action, readonly ActionGrant[]>>;
};

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
