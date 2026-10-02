import { projectsCatalogue } from "./catalogue.deploy";
import { githubRows, limitsRows, rootKeyRows } from "./catalogue.rows";
import {
  ACTIONS,
  type ActionGrant,
  type CatalogueGroup,
  type PermissionRow,
  type ScopeCatalogue,
  instancePath,
} from "./catalogue.types";

// The workspace scope includes the project tree with the project left open and
// the workspace-level resources from the canonical URN catalog.
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
    { id: "root_keys", label: "Root keys", rows: rootKeyRows() },
    { id: "github", label: "Connections", rows: githubRows() },
    { id: "limits", label: "Limits", rows: limitsRows() },
  ],
};
