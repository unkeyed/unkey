import { CATALOGUES, catalogueRows } from "./catalogue";
import {
  ACTIONS,
  type Action,
  CRUD_ACTIONS,
  type PermissionRow,
  type PermissionSelection,
  READ_ACTIONS,
  type ResourceScope,
} from "./catalogue.types";
import { type Policy, newPolicy, setRowsActions } from "./policy";

export const TEMPLATE_IDS = [
  "read",
  "write",
  "compute-read",
  "compute-write",
  "verify",
  "ratelimit",
  "custom",
] as const;

export type TemplateId = (typeof TEMPLATE_IDS)[number];

export type RootKeyTemplate = {
  id: TemplateId;
  title: string;
  description: string;
  materialise: () => Policy[];
};

function everyRow(scope: ResourceScope, actions: readonly Action[]): Policy {
  const policy = newPolicy(scope);
  return {
    ...policy,
    selection: setRowsActions(policy.selection, catalogueRows(CATALOGUES[scope]), actions),
  };
}

const COMPUTE_GROUPS: ReadonlySet<string> = new Set([
  "projects",
  "apps",
  "environments",
  "deployments",
  "gateway",
  "github",
  "limits",
  "usage",
  "logs",
]);

function computeRows(): PermissionRow[] {
  return CATALOGUES.workspace.groups
    .filter((group) => COMPUTE_GROUPS.has(group.id))
    .flatMap((group) => group.rows);
}

function computePolicy(actions: readonly Action[]): Policy {
  const policy = newPolicy("workspace");
  return { ...policy, selection: setRowsActions(policy.selection, computeRows(), actions) };
}

function withSelection(scope: ResourceScope, selection: PermissionSelection): Policy {
  return { ...newPolicy(scope), selection };
}

export const TEMPLATES: readonly RootKeyTemplate[] = [
  {
    id: "read",
    title: "All read permissions",
    description: "Every resource, read only",
    materialise: () => [everyRow("workspace", READ_ACTIONS)],
  },
  {
    id: "write",
    title: "All write permissions",
    description: "Every resource, full control",
    materialise: () => [everyRow("workspace", ACTIONS)],
  },
  {
    id: "compute-read",
    title: "Compute read only",
    description: "Projects, apps, deployments, and logs, read only",
    materialise: () => [computePolicy(READ_ACTIONS)],
  },
  {
    id: "compute-write",
    title: "Compute full control",
    description: "Projects, apps, deployments, domains, and variables",
    materialise: () => [computePolicy(CRUD_ACTIONS)],
  },
  {
    id: "verify",
    title: "Verify keys",
    description: "Every keyspace, verify only",
    materialise: () => [withSelection("keyspaces", { key: ["verify"] })],
  },
  {
    id: "ratelimit",
    title: "Standalone ratelimiting",
    description: "Namespaces, logs, and overrides",
    materialise: () => [
      withSelection("ratelimit-namespaces", {
        ratelimit_namespace: ["limit"],
        ratelimit_log: ["read"],
        ratelimit_override: [...CRUD_ACTIONS],
      }),
    ],
  },
  {
    id: "custom",
    title: "Start new",
    description: "Build a custom policy",
    materialise: () => [newPolicy()],
  },
];
