export const tags = {
  api: (apiId: string): string => `api-${apiId}`,
  permission: (permissionId: string): string => `permission-${permissionId}`,
  namespace: (namespaceId: string): string => `namespace-${namespaceId}`,
  role: (roleId: string): string => `role-${roleId}`,
};
