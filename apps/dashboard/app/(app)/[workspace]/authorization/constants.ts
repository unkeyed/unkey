export const navigation = (workspaceSlug: string) => [
  {
    label: "Roles",
    href: `/${workspaceSlug}/authorization/roles`,
    segment: "roles",
  },
  {
    label: "Permissions",
    href: `/${workspaceSlug}/authorization/permissions`,
    segment: "permissions",
  },
];
