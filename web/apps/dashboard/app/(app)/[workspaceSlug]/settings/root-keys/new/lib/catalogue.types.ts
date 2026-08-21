// The three coarse actions below are a UI vocabulary. The Go action vocabulary
// (`pkg/rbac/permissions/*.go`) is still partial, so some catalogue rows expand
// into URNs the API cannot enforce yet.
export const ACTIONS = ["read", "write", "delete"] as const;

export type Action = (typeof ACTIONS)[number];

export const RESOURCE_SCOPES = ["workspace"] as const;

export type ResourceScope = (typeof RESOURCE_SCOPES)[number];

export type PermissionRow = {
  id: string;
  label: string;
  description: string;
  path: string;
  resource: string;
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
  groups: CatalogueGroup[];
};

export type PermissionSelection = Record<string, readonly Action[]>;
