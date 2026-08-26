import { projectsCatalogue } from "./catalogue.deploy";
import { githubRows } from "./catalogue.rows";
import {
  ACTIONS,
  type ActionGrant,
  type CatalogueGroup,
  type PermissionRow,
  type ScopeCatalogue,
  instancePath,
} from "./catalogue.types";

// Everything a root key can reach lives under a project, except the GitHub app
// the workspace connects. So the workspace scope is the project tree with the
// project left open, plus that one connection.
const WILDCARD = "*";

function wildcardGrant(grant: ActionGrant): ActionGrant {
  return { ...grant, path: instancePath(grant.path, WILDCARD) };
}

function wildcardRow(row: PermissionRow): PermissionRow {
  const actions = {} as PermissionRow["actions"];
  for (const action of ACTIONS) {
    actions[action] = row.actions[action].map(wildcardGrant);
  }
  return { ...row, path: instancePath(row.path, WILDCARD), actions };
}

function wildcardGroups(catalogue: ScopeCatalogue): CatalogueGroup[] {
  return catalogue.groups.map((group) => ({ ...group, rows: group.rows.map(wildcardRow) }));
}

export const workspaceCatalogue: ScopeCatalogue = {
  scope: "workspace",
  label: "Workspace",
  allLabel: "All resources",
  instanceNoun: null,
  allInstance: WILDCARD,
  groups: [
    ...wildcardGroups(projectsCatalogue),
    { id: "github", label: "Connections", rows: githubRows() },
  ],
};
