import { CATALOGUES, catalogueRows } from "./catalogue";
import {
  type Action,
  COARSE_ACTIONS,
  type PermissionSelection,
  type ResourceScope,
} from "./catalogue.types";
import { type Policy, newPolicy, setRowsActions } from "./policy";

export const TEMPLATE_IDS = ["read", "write", "verify", "ratelimit", "custom"] as const;

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

function withSelection(scope: ResourceScope, selection: PermissionSelection): Policy {
  return { ...newPolicy(scope), selection };
}

export const TEMPLATES: readonly RootKeyTemplate[] = [
  {
    id: "read",
    title: "All read permissions",
    description: "Every resource, read only",
    materialise: () => [everyRow("workspace", ["read"])],
  },
  {
    id: "write",
    title: "All write permissions",
    description: "Every resource, full control",
    materialise: () => [everyRow("workspace", COARSE_ACTIONS)],
  },
  {
    id: "verify",
    title: "Verify keys",
    description: "Every keyspace, verify only",
    materialise: () => [withSelection("keyspaces", { key: ["read"] })],
  },
  {
    id: "ratelimit",
    title: "Standalone ratelimiting",
    description: "Namespaces and overrides",
    materialise: () => [
      withSelection("ratelimit-namespaces", { namespace: ["read"], override: [...COARSE_ACTIONS] }),
    ],
  },
  {
    id: "custom",
    title: "Start new",
    description: "Build a custom policy",
    materialise: () => [newPolicy()],
  },
];
