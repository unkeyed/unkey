const keysRoot = ["keys"] as const;
const rbacRoot = ["rbac"] as const;
const rolesRoot = [...rbacRoot, "roles"] as const;
const permissionsRoot = [...rbacRoot, "permissions"] as const;

export const queryKeys = {
  keys: {
    all: keysRoot,
    detail: (keyId: string) => [...keysRoot, "detail", keyId] as const,
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
};
