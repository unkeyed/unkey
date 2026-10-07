import {
  appRow,
  deploymentRows,
  environmentRows,
  gatewayRows,
  identityRows,
  keyspaceRows,
  namespaceRows,
  projectRow,
  rbacRows,
} from "./catalogue.rows";
import { type CatalogueGroup, INSTANCE_TOKEN, type ScopeCatalogue } from "./catalogue.types";

export type DeployScope = "projects" | "apps" | "environments";

// The instance a policy names is the whole path down to it, not a bare id: an
// app only exists inside a project, and a grant that says otherwise ("this app
// in any project") points at nothing the API can resolve. Each scope therefore
// anchors the tree at its own level, wildcards everything below it, and offers
// only the resources that live under that anchor.
type DeploySpec = {
  label: string;
  allLabel: string;
  instanceNoun: string;
  allInstance: string;
  project: string | null;
  app: string | null;
  environment: string;
};

const PROJECT_PATH = `projects/${INSTANCE_TOKEN}`;
const APP_UNDER_PROJECT = `${PROJECT_PATH}/apps/*`;

const DEPLOY_SPECS: Record<DeployScope, DeploySpec> = {
  projects: {
    label: "Projects",
    allLabel: "All projects",
    instanceNoun: "projects",
    allInstance: "*",
    project: PROJECT_PATH,
    app: APP_UNDER_PROJECT,
    environment: `${APP_UNDER_PROJECT}/environments/*`,
  },
  apps: {
    label: "Apps",
    allLabel: "All apps",
    instanceNoun: "apps",
    allInstance: "projects/*/apps/*",
    project: null,
    app: INSTANCE_TOKEN,
    environment: `${INSTANCE_TOKEN}/environments/*`,
  },
  environments: {
    label: "Environments",
    allLabel: "All environments",
    instanceNoun: "environments",
    allInstance: "projects/*/apps/*/environments/*",
    project: null,
    app: null,
    environment: INSTANCE_TOKEN,
  },
};

function deployGroups({ project, app, environment }: DeploySpec): CatalogueGroup[] {
  const groups: CatalogueGroup[] = [];

  if (project !== null) {
    groups.push({ id: "projects", label: "Projects", rows: [projectRow(project)] });
  }
  if (app !== null) {
    groups.push({ id: "apps", label: "Apps", rows: [appRow(app)] });
  }
  groups.push(
    { id: "environments", label: "Environments", rows: environmentRows(environment) },
    { id: "deployments", label: "Deployments", rows: deploymentRows(environment) },
    { id: "gateway", label: "Gateway", rows: gatewayRows(environment) },
  );
  if (project !== null) {
    groups.push(
      { id: "keyspaces", label: "Key management", rows: keyspaceRows(`${project}/keyspaces/*`) },
      {
        id: "ratelimits",
        label: "Rate limiting",
        rows: namespaceRows(`${project}/ratelimits/namespaces/*`),
      },
      {
        id: "access",
        label: "Identity and RBAC",
        rows: [...identityRows(project), ...rbacRows(project)],
      },
    );
  }
  return groups;
}

export function deployCatalogue(scope: DeployScope): ScopeCatalogue {
  const spec = DEPLOY_SPECS[scope];
  return {
    scope,
    label: spec.label,
    allLabel: spec.allLabel,
    instanceNoun: spec.instanceNoun,
    allInstance: spec.allInstance,
    groups: deployGroups(spec),
  };
}

export const projectsCatalogue = deployCatalogue("projects");
export const appsCatalogue = deployCatalogue("apps");
export const environmentsCatalogue = deployCatalogue("environments");
