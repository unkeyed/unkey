import {
  type Action,
  type CatalogueGroup,
  type GrantSpec,
  INSTANCE_TOKEN,
  type PermissionRow,
  type ScopeCatalogue,
  permissionRow,
} from "./catalogue.types";

export type DeployScope = "projects" | "apps" | "environments";

type DeployLevel = {
  scope: DeployScope;
  segment: string;
  rowId: string;
  resource: string;
  label: string;
  ownLabel: string;
  allLabel: string;
  instanceNoun: string;
  extraWrite: readonly string[];
};

// `onEnvironment` lifts a grant off the collection the row stands for and onto
// the environment: no member of the collection exists until it is created.
type LeafGrant = {
  name: string;
  onEnvironment?: boolean;
};

type DeployLeaf = {
  rowId: string;
  label: string;
  resource: string;
  segment: string | null;
  actions: Partial<Record<Action, readonly LeafGrant[]>>;
};

const DEPLOY_LEVELS: readonly DeployLevel[] = [
  {
    scope: "projects",
    segment: "projects",
    rowId: "project",
    resource: "project",
    label: "Projects",
    ownLabel: "Project settings",
    allLabel: "All projects",
    instanceNoun: "projects",
    extraWrite: ["update_project"],
  },
  {
    scope: "apps",
    segment: "apps",
    rowId: "app",
    resource: "app",
    label: "Apps",
    ownLabel: "App settings",
    allLabel: "All apps",
    instanceNoun: "apps",
    extraWrite: ["update_app", "connect_repository"],
  },
  {
    scope: "environments",
    segment: "environments",
    rowId: "environment",
    resource: "environment",
    label: "Environments",
    ownLabel: "Environment settings",
    allLabel: "All environments",
    instanceNoun: "environments",
    extraWrite: ["update_environment"],
  },
];

const DEPLOY_LEAVES: readonly DeployLeaf[] = [
  {
    rowId: "deployment",
    label: "Deployments",
    resource: "deployment",
    segment: "deployments",
    actions: {
      write: [
        { name: "create_deployment" },
        { name: "start_deployment" },
        { name: "stop_deployment" },
        { name: "promote_deployment" },
        { name: "rollback_deployment" },
      ],
    },
  },
  {
    rowId: "domain",
    label: "Custom domains",
    resource: "domain",
    segment: "domains",
    actions: {
      write: [
        { name: "create_domain", onEnvironment: true },
        { name: "verify_domain", onEnvironment: true },
      ],
    },
  },
  {
    rowId: "variable",
    label: "Environment variables",
    resource: "variable",
    segment: null,
    actions: {
      read: [{ name: "read_environment_variables" }],
      write: [{ name: "create_variable" }, { name: "set_environment_variables" }],
      delete: [{ name: "remove_environment_variables" }],
    },
  },
];

function levelPaths(depth: number): string[] {
  const paths: string[] = [];
  for (const [index, level] of DEPLOY_LEVELS.entries()) {
    const parent = index === 0 ? "" : `${paths[index - 1]}/`;
    paths.push(`${parent}${level.segment}/${index === depth ? INSTANCE_TOKEN : "*"}`);
  }
  return paths;
}

function levelRow(depth: number, index: number, paths: readonly string[]): PermissionRow {
  const level = DEPLOY_LEVELS[index];
  const own = index === depth;
  const write: GrantSpec[] = own
    ? []
    : [{ name: `create_${level.resource}`, path: paths[index - 1] }];
  return permissionRow({
    id: level.rowId,
    label: own ? level.ownLabel : level.label,
    path: paths[index],
    resource: level.resource,
    actions: { write: [...write, ...level.extraWrite.map((name) => ({ name }))] },
  });
}

function leafRow(leaf: DeployLeaf, environmentPath: string): PermissionRow {
  const path = leaf.segment === null ? environmentPath : `${environmentPath}/${leaf.segment}/*`;
  const actions: Partial<Record<Action, readonly GrantSpec[]>> = {};
  for (const [action, grants] of Object.entries(leaf.actions) as [Action, LeafGrant[]][]) {
    actions[action] = grants.map((grant) =>
      grant.onEnvironment ? { name: grant.name, path: environmentPath } : { name: grant.name },
    );
  }
  return permissionRow({
    id: leaf.rowId,
    label: leaf.label,
    path,
    resource: leaf.resource,
    actions,
  });
}

export function deployCatalogue(scope: DeployScope): ScopeCatalogue {
  const depth = DEPLOY_LEVELS.findIndex((level) => level.scope === scope);
  const paths = levelPaths(depth);
  const environmentPath = paths[DEPLOY_LEVELS.length - 1];

  const groups: CatalogueGroup[] = DEPLOY_LEVELS.slice(depth).map((level, offset) => {
    const index = depth + offset;
    const rows = [levelRow(depth, index, paths)];
    if (index === DEPLOY_LEVELS.length - 1) {
      rows.push(...DEPLOY_LEAVES.map((leaf) => leafRow(leaf, environmentPath)));
    }
    return { id: level.scope, label: level.label, rows };
  });

  const picked = DEPLOY_LEVELS[depth];
  return {
    scope,
    label: picked.label,
    allLabel: picked.allLabel,
    instanceNoun: picked.instanceNoun,
    groups,
  };
}

export const projectsCatalogue = deployCatalogue("projects");
export const appsCatalogue = deployCatalogue("apps");
export const environmentsCatalogue = deployCatalogue("environments");
