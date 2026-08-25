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

const PROJECT_ROWS = ["project"];
const APP_ROWS = ["app"];

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

const wildcardRows = (catalogue: ScopeCatalogue): PermissionRow[] =>
  catalogue.groups.flatMap((group) => group.rows).map(wildcardRow);

const wildcardGroups = (catalogue: ScopeCatalogue): CatalogueGroup[] =>
  catalogue.groups.map((group) => ({
    ...group,
    label: catalogue.label,
    rows: group.rows.map(wildcardRow),
  }));

const deployRows = wildcardRows(projectsCatalogue);

const deployGroups: CatalogueGroup[] = [
  {
    id: "projects",
    label: "Projects",
    rows: deployRows.filter((row) => PROJECT_ROWS.includes(row.id)),
  },
  {
    id: "apps",
    label: "Apps",
    rows: deployRows.filter((row) => APP_ROWS.includes(row.id)),
  },
  {
    id: "environments",
    label: "Environments",
    rows: deployRows.filter((row) => !PROJECT_ROWS.includes(row.id) && !APP_ROWS.includes(row.id)),
  },
];

export const workspaceCatalogue: ScopeCatalogue = {
  scope: "workspace",
  label: "Workspace",
  allLabel: "All resources",
  instanceNoun: null,
  groups: [
    ...deployGroups,
    ...wildcardGroups(keyspacesCatalogue),
    ...wildcardGroups(ratelimitNamespacesCatalogue),
    ...identitiesCatalogue.groups,
    ...rbacCatalogue.groups,
    ...vaultCatalogue.groups,
  ],
};
