import { identitiesCatalogue } from "./catalogue.identities";
import { keyspacesCatalogue } from "./catalogue.keyspaces";
import { projectsCatalogue } from "./catalogue.projects";
import { ratelimitNamespacesCatalogue } from "./catalogue.ratelimit-namespaces";
import { rbacCatalogue } from "./catalogue.rbac";
import {
  ACTIONS,
  type Action,
  type ActionGrant,
  type CatalogueGroup,
  INSTANCE_TOKEN,
  type PermissionRow,
  type ScopeCatalogue,
} from "./catalogue.types";
import { vaultCatalogue } from "./catalogue.vault";

const wildcardPath = (path: string) => path.split(INSTANCE_TOKEN).join("*");

const wildcardGrant = (grant: ActionGrant): ActionGrant =>
  grant.path === undefined ? grant : { ...grant, path: wildcardPath(grant.path) };

function wildcardActions(
  actions: NonNullable<PermissionRow["actions"]>,
): NonNullable<PermissionRow["actions"]> {
  const next: Partial<Record<Action, ActionGrant[]>> = {};
  for (const action of ACTIONS) {
    const grants = actions[action];
    if (grants) {
      next[action] = grants.map(wildcardGrant);
    }
  }
  return next;
}

const wildcardRow = (row: PermissionRow): PermissionRow => ({
  ...row,
  path: wildcardPath(row.path),
  actions: row.actions === undefined ? undefined : wildcardActions(row.actions),
});

const wildcardGroups = (catalogue: ScopeCatalogue): CatalogueGroup[] =>
  catalogue.groups.map((group) => ({
    ...group,
    label: catalogue.label,
    rows: group.rows.map(wildcardRow),
  }));

export const workspaceCatalogue: ScopeCatalogue = {
  scope: "workspace",
  label: "Workspace",
  allLabel: "All resources",
  instanceNoun: null,
  groups: [
    ...wildcardGroups(projectsCatalogue),
    ...wildcardGroups(keyspacesCatalogue),
    ...wildcardGroups(ratelimitNamespacesCatalogue),
    ...identitiesCatalogue.groups,
    ...rbacCatalogue.groups,
    ...vaultCatalogue.groups,
  ],
};
