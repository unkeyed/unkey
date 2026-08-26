import { type ScopeCatalogue, permissionRow } from "./catalogue.types";

export const rbacCatalogue: ScopeCatalogue = {
  scope: "rbac",
  label: "RBAC",
  allLabel: "All roles and permissions",
  instanceNoun: null,
  groups: [
    {
      id: "rbac",
      label: "RBAC",
      rows: [
        permissionRow({
          id: "role",
          label: "Role definitions",
          path: "rbac/roles/*",
          resource: "role",
        }),
        permissionRow({
          id: "permission",
          label: "Permission definitions",
          path: "rbac/permissions/*",
          resource: "permission",
        }),
      ],
    },
  ],
};
