import type { ScopeCatalogue } from "./catalogue.types";

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
        {
          id: "role",
          label: "Role definitions",
          path: "rbac/roles/*",
          resource: "role",
        },
        {
          id: "permission",
          label: "Permission definitions",
          path: "rbac/permissions/*",
          resource: "permission",
        },
      ],
    },
  ],
};
