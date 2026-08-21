import type { PermissionRow, ResourceScope, ScopeCatalogue } from "./catalogue.types";
import { workspaceCatalogue } from "./catalogue.workspace";

export const CATALOGUES: Record<ResourceScope, ScopeCatalogue> = {
  workspace: workspaceCatalogue,
};

export function catalogueFor(scope: ResourceScope): ScopeCatalogue {
  return CATALOGUES[scope];
}

export function catalogueRows(catalogue: ScopeCatalogue): PermissionRow[] {
  return catalogue.groups.flatMap((group) => group.rows);
}
