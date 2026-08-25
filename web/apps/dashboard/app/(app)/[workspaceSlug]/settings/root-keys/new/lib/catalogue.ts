import { appsCatalogue } from "./catalogue.apps";
import { environmentsCatalogue } from "./catalogue.environments";
import { identitiesCatalogue } from "./catalogue.identities";
import { keyspacesCatalogue } from "./catalogue.keyspaces";
import { projectsCatalogue } from "./catalogue.projects";
import { ratelimitNamespacesCatalogue } from "./catalogue.ratelimit-namespaces";
import { rbacCatalogue } from "./catalogue.rbac";
import type { PermissionRow, ResourceScope, ScopeCatalogue } from "./catalogue.types";
import { vaultCatalogue } from "./catalogue.vault";
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
  vault: vaultCatalogue,
};

export function catalogueFor(scope: ResourceScope): ScopeCatalogue {
  return CATALOGUES[scope];
}

export function catalogueRows(catalogue: ScopeCatalogue): PermissionRow[] {
  return catalogue.groups.flatMap((group) => group.rows);
}
