import { projectsCatalogue } from "./catalogue.projects";
import {
  ACTIONS,
  type Action,
  type ActionGrant,
  type CatalogueGroup,
  INSTANCE_TOKEN,
  type PermissionRow,
  type ScopeCatalogue,
  actionGrant,
} from "./catalogue.types";

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
    rows: group.rows.map(wildcardRow),
  }));

export const workspaceCatalogue: ScopeCatalogue = {
  scope: "workspace",
  label: "Entire workspace",
  allLabel: "All resources",
  instanceNoun: null,
  groups: [
    ...wildcardGroups(projectsCatalogue),
    {
      id: "github",
      label: "Connections",
      rows: [
        {
          id: "github_app",
          label: "GitHub apps",
          path: "github/apps/*",
          resource: "github_app",
          actions: {
            read_github_app: actionGrant("read_github_app"),
            write_github_app: actionGrant("write_github_app"),
            delete_github_app: actionGrant("delete_github_app"),
          },
        },
      ],
    },
  ],
};
