import type { EnvironmentKind } from "@/lib/queries/environments";
import {
  CLUSTERS,
  type Cluster,
  type Compute,
  type CreateDatabaseInput,
  type Database,
  type Engine,
  type EnvironmentBranches,
  type PlanetscaleDatabase,
  type PlanetscaleOrg,
  type RegionId,
  environmentBranch,
  orgHasCard,
  skusFor,
} from "../database-model";

const STEPS = ["connect", "source", "database", "environments", "review"] as const;

export type Step = (typeof STEPS)[number];

export const SOURCE_CHOICES = ["new", "existing"] as const;

export type SourceChoice = (typeof SOURCE_CHOICES)[number];

export type SourceStep = Exclude<Step, "connect" | "source">;

export const showsAccount: Record<Step, boolean> = {
  connect: false,
  source: true,
  database: true,
  environments: false,
  review: false,
};

export const stepCopy: Record<Step, { nav: string; title: string; description: string }> = {
  connect: {
    nav: "Connect",
    title: "Connect PlanetScale",
    description:
      "Your databases stay in your own PlanetScale account. Unkey links them to this project, and PlanetScale bills you directly.",
  },
  source: {
    nav: "Source",
    title: "Choose a database",
    description: "Create a new PlanetScale database, or link one you already have.",
  },
  database: {
    nav: "Configure",
    title: "Configure database",
    description: "Unkey creates the database in your PlanetScale organization.",
  },
  environments: {
    nav: "Environments",
    title: "Production and preview",
    description: "Size your production database. Preview deployments get their own branch.",
  },
  review: {
    nav: "Review",
    title: "Review",
    description: "Check the details before you continue.",
  },
};

export type EnvironmentPlan = "production_preview" | "production";

const planCopy: Record<EnvironmentPlan, string> = {
  production_preview: "Production + Preview",
  production: "Production",
};

const environmentsFor: Record<EnvironmentPlan, EnvironmentKind[]> = {
  production_preview: ["production", "preview"],
  production: ["production"],
};

export const branchFor: Record<EnvironmentKind, string> = {
  production: "main",
  preview: "preview",
};

const previewSharesWith: Record<EnvironmentPlan, EnvironmentKind> = {
  production_preview: "preview",
  production: "production",
};

function forPlan<T>(
  plan: EnvironmentPlan,
  values: Record<EnvironmentKind, T>,
): Record<EnvironmentKind, T> {
  return { production: values.production, preview: values[previewSharesWith[plan]] };
}

export type DatabaseConfig = {
  name: string;
  engine: Engine;
  region: RegionId;
};

type FlowState = {
  step: Step;
  source: SourceChoice | null;
  org: string | null;
  plan: EnvironmentPlan;
  config: DatabaseConfig;
  compute: Record<EnvironmentKind, Compute>;
  selection: { database: string; branches: Record<EnvironmentKind, string> | null } | null;
};

export function clusterAvailable(engine: Engine, cluster: Cluster): boolean {
  return skusFor(engine, cluster).length > 0;
}

function reconcileCompute(engine: Engine, compute: Compute): Compute {
  const cluster = clusterAvailable(engine, compute.cluster)
    ? compute.cluster
    : (CLUSTERS.find((c) => clusterAvailable(engine, c)) ?? compute.cluster);
  const skus = skusFor(engine, cluster);
  const size = skus.some((sku) => sku.id === compute.size) ? compute.size : skus[0].id;
  return { cluster, size };
}

export const initialFlow: FlowState = {
  step: "connect",
  source: null,
  org: null,
  plan: "production_preview",
  config: { name: "", engine: "postgres", region: "us-east-2" },
  compute: {
    production: { cluster: "single", size: "PS-5" },
    preview: { cluster: "single", size: "PS-5" },
  },
  selection: null,
};

export function monthlyPrice(engine: Engine, compute: Compute): number {
  return skusFor(engine, compute.cluster).find((sku) => sku.id === compute.size)?.monthlyPrice ?? 0;
}

const DATABASE_NAME = /^[a-z0-9][a-z0-9_-]*$/;

function databaseNameError(
  orgDatabases: PlanetscaleDatabase[],
  org: string,
  name: string,
): string | null {
  if (name === "") {
    return null;
  }
  if (!DATABASE_NAME.test(name)) {
    return "Use lowercase letters, numbers, - and _. Start with a letter or number.";
  }
  if (orgDatabases.some((db) => db.name === name)) {
    return `${org} already has a database named ${name}.`;
  }
  return null;
}

function defaultPreview(db: PlanetscaleDatabase, production: string): string {
  const others = db.branches.filter((branch) => branch !== production);
  return others.find((branch) => branch === branchFor.preview) ?? others[0] ?? branchFor.preview;
}

export function branchOptions(
  db: PlanetscaleDatabase,
  production: string,
): Record<EnvironmentKind, { value: string; label: string }[]> {
  const preview = db.branches.filter((branch) => branch !== production);
  const create = db.branches.includes(branchFor.preview)
    ? []
    : [{ value: branchFor.preview, label: "Create a preview branch" }];
  return {
    production: db.branches.map((branch) => ({ value: branch, label: branch })),
    preview: [...preview.map((branch) => ({ value: branch, label: branch })), ...create],
  };
}

export function withBranch(
  db: PlanetscaleDatabase,
  branches: Record<EnvironmentKind, string>,
  env: EnvironmentKind,
  branch: string,
): Record<EnvironmentKind, string> {
  const next = { ...branches, [env]: branch };
  return next.preview === next.production
    ? { ...next, preview: defaultPreview(db, next.production) }
    : next;
}

type FlowAction =
  | { type: "step"; step: Step }
  | { type: "source"; source: SourceChoice }
  | { type: "org"; org: string }
  | { type: "config"; config: DatabaseConfig }
  | { type: "compute"; env: EnvironmentKind; compute: Compute }
  | { type: "plan"; plan: EnvironmentPlan }
  | { type: "select"; database: string }
  | { type: "branches"; branches: Record<EnvironmentKind, string> };

export function flowReducer(state: FlowState, action: FlowAction): FlowState {
  switch (action.type) {
    case "step":
      return { ...state, step: action.step };
    case "source":
      return { ...state, source: action.source };
    case "org":
      return { ...state, org: action.org, selection: null };
    case "config":
      return {
        ...state,
        config: action.config,
        compute: {
          production: reconcileCompute(action.config.engine, state.compute.production),
          preview: reconcileCompute(action.config.engine, state.compute.preview),
        },
      };
    case "compute":
      return { ...state, compute: { ...state.compute, [action.env]: action.compute } };
    case "plan":
      return { ...state, plan: action.plan };
    case "select":
      return { ...state, selection: { database: action.database, branches: null } };
    case "branches":
      return {
        ...state,
        selection: state.selection && { ...state.selection, branches: action.branches },
      };
  }
}

function existingView(orgDatabases: PlanetscaleDatabase[], { selection, plan }: FlowState) {
  const database = orgDatabases.find((db) => db.name === selection?.database);
  if (!database) {
    return null;
  }
  const branches = selection?.branches ?? {
    production: database.defaultBranch,
    preview: defaultPreview(database, database.defaultBranch),
  };
  return {
    database,
    branches,
    linkedBranches: forPlan(plan, {
      production: environmentBranch(database, branches.production),
      preview: environmentBranch(database, branches.preview),
    }),
  };
}

type Facts = {
  org: string;
  nameValid: boolean;
  hasCard: boolean;
  newBranches: EnvironmentBranches;
  existing: ReturnType<typeof existingView>;
};

export const sourceFlow: Record<
  SourceChoice,
  {
    label: string;
    hint: string;
    submit: string;
    previewConfirm: string;
    copy: Partial<Record<Step, Partial<Record<"title" | "description", string>>>>;
    detail: (state: FlowState, facts: Facts) => string | null;
    input: (state: FlowState, facts: Facts) => CreateDatabaseInput | null;
  }
> = {
  new: {
    label: "New database",
    hint: "Create a database on PlanetScale from here.",
    submit: "Create database",
    previewConfirm: "Don't create preview branch",
    copy: {},
    detail: ({ config }) => (config.name ? `${config.name} · ${config.region}` : null),
    input: ({ config, compute }, { org, nameValid, hasCard, newBranches }) =>
      nameValid && hasCard
        ? { source: "new", org, ...config, compute: compute.production, branches: newBranches }
        : null,
  },
  existing: {
    label: "Existing database",
    hint: "Link a database you already have on PlanetScale.",
    submit: "Add database",
    previewConfirm: "Don't use preview branch",
    copy: {
      database: {
        title: "Select database",
        description: "Pick the PlanetScale database to link to this project.",
      },
      environments: {
        description: "Pick the branches production and preview deployments use.",
      },
    },
    detail: (_, { existing }) => existing?.database.name ?? null,
    input: (_, { existing }) =>
      existing && {
        source: "existing",
        org: existing.database.org,
        name: existing.database.name,
        branches: existing.linkedBranches,
      },
  },
};

type FlowSources = {
  connected: boolean;
  orgs: PlanetscaleOrg[];
  databases: PlanetscaleDatabase[];
  linked: Database[];
};

function connectedFlow(state: FlowState, { orgs, databases, linked }: FlowSources) {
  const org = state.org ?? orgs[0].name;
  const orgDatabases = databases.filter((db) => db.org === org);
  const nameError = databaseNameError(orgDatabases, org, state.config.name);
  const facts: Facts = {
    org,
    nameValid: state.config.name !== "" && nameError === null,
    hasCard: orgHasCard(orgs, org),
    newBranches: forPlan(state.plan, {
      production: { branch: branchFor.production, compute: state.compute.production },
      preview: { branch: branchFor.preview, compute: state.compute.preview },
    }),
    existing: existingView(orgDatabases, state),
  };
  const source = state.source && sourceFlow[state.source];
  return {
    ...facts,
    connected: true as const,
    orgs,
    nameError,
    total: environmentsFor[state.plan].reduce(
      (total, env) => total + monthlyPrice(state.config.engine, state.compute[env]),
      0,
    ),
    linkable: orgDatabases.filter(
      (db) => !linked.some((l) => l.planetscale.org === db.org && l.name === db.name),
    ),
    details: source && {
      connect: null,
      source: `${source.label} · ${org}`,
      database: source.detail(state, facts),
      environments: planCopy[state.plan],
      review: null,
    },
    input: source?.input(state, facts) ?? null,
  };
}

export function deriveFlow(state: FlowState, sources: FlowSources) {
  const steps: Step[] = sources.connected ? STEPS.slice(1) : [...STEPS];
  const step = sources.connected && steps.includes(state.step) ? state.step : steps[0];
  const at = steps.indexOf(step);
  const nav = {
    steps,
    step,
    at,
    order: STEPS.indexOf(step),
    previous: steps[at - 1] ?? null,
    next: steps[at + 1] ?? null,
  };
  return sources.connected
    ? { ...nav, ...connectedFlow(state, sources) }
    : { ...nav, connected: false as const, details: null };
}

export type ConnectedFlowView = Extract<ReturnType<typeof deriveFlow>, { connected: true }>;
