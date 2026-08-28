import { appsCatalogue, environmentsCatalogue, projectsCatalogue } from "./catalogue.deploy";
import { identitiesCatalogue } from "./catalogue.identities";
import { keyspacesCatalogue } from "./catalogue.keyspaces";
import { ratelimitNamespacesCatalogue } from "./catalogue.ratelimit-namespaces";
import { rbacCatalogue } from "./catalogue.rbac";
import type { PermissionRow, ResourceScope, ScopeCatalogue } from "./catalogue.types";
import { workspaceCatalogue } from "./catalogue.workspace";

export const CATALOGUES: Record<ResourceScope, ScopeCatalogue> = {
  workspace: workspaceCatalogue,
  projects: projectsCatalogue,
  apps: appsCatalogue,
  environments: environmentsCatalogue,
  keyspaces: keyspacesCatalogue,
  "ratelimit-namespaces": ratelimitNamespacesCatalogue,
  identities: identitiesCatalogue,
  rbac: rbacCatalogue,
};

export function catalogueRows(catalogue: ScopeCatalogue): PermissionRow[] {
  return catalogue.groups.flatMap((group) => group.rows);
}
