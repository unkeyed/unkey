import { catalogueFor, catalogueRows } from "./catalogue";
import {
  ACTIONS,
  type Action,
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

const BREADTH_SCOPES: readonly ResourceScope[] = ["workspace"];

function everyRow(scope: ResourceScope, actions: readonly Action[]): Policy {
  const policy = newPolicy(scope);
  return {
    ...policy,
    selection: setRowsActions(policy.selection, catalogueRows(catalogueFor(scope)), actions),
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
    materialise: () =>
      BREADTH_SCOPES.map((scope) =>
        everyRow(scope, [
          "read_project",
          "read_app",
          "read_environment",
          "read_deployment",
          "read_deployment_logs",
          "read_domain",
          "read_environment_variable",
          "read_gateway_logs",
          "read_gateway_policy",
          "read_identity",
          "read_keyspace",
          "read_keyspace_logs",
          "read_key",
          "read_ratelimit_namespace",
          "read_ratelimit_logs",
          "read_ratelimit_override",
          "read_role",
          "read_permission",
          "read_github_app",
        ]),
      ),
  },
  {
    id: "write",
    title: "All write permissions",
    description: "Every resource, full control",
    materialise: () => BREADTH_SCOPES.map((scope) => everyRow(scope, ACTIONS)),
  },
  {
    id: "verify",
    title: "Verify keys",
    description: "Every keyspace, verify only",
    materialise: () => [withSelection("keyspaces", { key: ["verify_key"] })],
  },
  {
    id: "ratelimit",
    title: "Standalone ratelimiting",
    description: "Namespaces, logs, and overrides",
    materialise: () => [
      withSelection("ratelimit-namespaces", {
        ratelimit_namespace: ["limit_ratelimit_namespace"],
        ratelimit_log: ["read_ratelimit_logs"],
        ratelimit_override: [
          "read_ratelimit_override",
          "write_ratelimit_override",
          "delete_ratelimit_override",
        ],
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
