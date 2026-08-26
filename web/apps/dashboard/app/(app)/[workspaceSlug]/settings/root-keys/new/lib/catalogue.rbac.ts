import { type ScopeCatalogue, actionGrant } from "./catalogue.types";

export const rbacCatalogue: ScopeCatalogue = {
  scope: "rbac",
  label: "All roles and permissions",
  allLabel: "All roles and permissions",
  instanceNoun: null,
  groups: [
    {
      id: "rbac",
      label: "Roles and permissions",
      rows: [
        {
          id: "role",
          label: "Roles",
          path: "projects/*/rbac/roles/*",
          resource: "role",
          actions: {
            read_role: actionGrant("read_role", "roles:read"),
            write_role: actionGrant("write_role", "roles:write"),
            delete_role: actionGrant("delete_role", "roles:delete"),
          },
        },
        {
          id: "permission",
          label: "Permissions",
          path: "projects/*/rbac/permissions/*",
          resource: "permission",
          actions: {
            read_permission: actionGrant("read_permission", "permissions:read"),
            write_permission: actionGrant("write_permission", "permissions:write"),
            delete_permission: actionGrant("delete_permission", "permissions:delete"),
          },
        },
      ],
    },
  ],
};
