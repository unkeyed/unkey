import { projectsCatalogue } from "./catalogue.deploy";
import { identitiesCatalogue } from "./catalogue.identities";
import { keyspacesCatalogue } from "./catalogue.keyspaces";
import { ratelimitNamespacesCatalogue } from "./catalogue.ratelimit-namespaces";
import { rbacCatalogue } from "./catalogue.rbac";
import {
  ACTIONS,
  ALL_INSTANCES,
  type ActionGrant,
  type CatalogueGroup,
  type PermissionRow,
  type ScopeCatalogue,
  instancePath,
} from "./catalogue.types";

const PARTITIONED: readonly ScopeCatalogue[] = [
  projectsCatalogue,
  keyspacesCatalogue,
  ratelimitNamespacesCatalogue,
  identitiesCatalogue,
  rbacCatalogue,
];

function wildcardGrant(grant: ActionGrant): ActionGrant {
  return { ...grant, path: instancePath(grant.path, ALL_INSTANCES) };
}

function wildcardRow(row: PermissionRow): PermissionRow {
  const actions = {} as PermissionRow["actions"];
  for (const action of ACTIONS) {
    actions[action] = row.actions[action].map(wildcardGrant);
  }
  return { ...row, path: instancePath(row.path, ALL_INSTANCES), actions };
}

function wildcardGroups(catalogue: ScopeCatalogue): CatalogueGroup[] {
  return catalogue.groups.map((group) => ({ ...group, rows: group.rows.map(wildcardRow) }));
}

export const workspaceCatalogue: ScopeCatalogue = {
  scope: "workspace",
  label: "Workspace",
  allLabel: "All resources",
  instanceNoun: null,
  groups: PARTITIONED.flatMap(wildcardGroups),
};
