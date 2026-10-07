import { rbacRows } from "./catalogue.rows";
import type { ScopeCatalogue } from "./catalogue.types";

export const rbacCatalogue: ScopeCatalogue = {
  scope: "rbac",
  label: "RBAC",
  allLabel: "All roles and permissions",
  instanceNoun: null,
  allInstance: "*",
  groups: [{ id: "rbac", label: "Roles and permissions", rows: rbacRows("projects/*") }],
};
