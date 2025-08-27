export const navigation = (workspaceId: string) => [
  {
    label: "Roles",
    href: `/${workspaceId}/authorization/roles`,
    segment: "roles",
  },
  {
    label: "Permissions",
    href: `/${workspaceId}/authorization/permissions`,
    segment: "permissions",
  },
];
