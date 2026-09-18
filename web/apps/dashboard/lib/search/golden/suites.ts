import type { SearchSpec } from "@/lib/search/spec";
import { auditSearchSpec } from "@/lib/search/specs/audit";
import { deploymentsSearchSpec } from "@/lib/search/specs/deployments";
import { keysListSearchSpec } from "@/lib/search/specs/keys-list";
import { keysOverviewSearchSpec } from "@/lib/search/specs/keys-overview";
import { logsSearchSpec } from "@/lib/search/specs/logs";
import { permissionsSearchSpec } from "@/lib/search/specs/permissions";
import { ratelimitSearchSpec } from "@/lib/search/specs/ratelimit";
import { requestLogsSearchSpec } from "@/lib/search/specs/request-logs";
import { rolesSearchSpec } from "@/lib/search/specs/roles";
import { rootKeysSearchSpec } from "@/lib/search/specs/root-keys";
import { runtimeLogsSearchSpec } from "@/lib/search/specs/runtime-logs";
import { GOLDEN_QUERIES, type GoldenQuery } from "./queries";

export type Suite = {
  name: string;
  spec: SearchSpec;
  queries: GoldenQuery[];
};

export const SUITES: Suite[] = [
  {
    name: "logs",
    spec: logsSearchSpec,
    queries: GOLDEN_QUERIES,
  },
  {
    name: "request logs",
    spec: requestLogsSearchSpec,
    queries: [
      {
        query: "failed requests in the last 2h",
        expected: [
          { field: "status", operator: "is", value: 400 },
          { field: "status", operator: "is", value: 500 },
          { field: "since", operator: "is", value: "2h" },
        ],
      },
      {
        query: "POST requests to /v1/keys",
        expected: [
          { field: "methods", operator: "is", value: "POST" },
          { field: "paths", operator: "startsWith", value: "v1/keys" },
        ],
      },
      {
        query: "requests served from iad1",
        expected: [{ field: "region", operator: "is", value: "iad1" }],
      },
      {
        query: "everything from api.unkey.dev in the last 24h",
        expected: [
          { field: "host", operator: "is", value: "api.unkey.dev" },
          { field: "since", operator: "is", value: "24h" },
        ],
      },
      {
        query: "server errors from the past 7 days",
        expected: [
          { field: "status", operator: "is", value: 500 },
          { field: "since", operator: "is", value: "7d" },
        ],
      },
    ],
  },
  {
    name: "runtime logs",
    spec: runtimeLogsSearchSpec,
    queries: [
      {
        query: "errors in the last hour",
        expected: [
          { field: "severity", operator: "is", value: "ERROR" },
          { field: "since", operator: "is", value: "1h" },
        ],
      },
      {
        query: "warnings and errors from the past 3 days",
        expected: [
          { field: "severity", operator: "is", value: "ERROR" },
          { field: "severity", operator: "is", value: "WARN" },
          { field: "since", operator: "is", value: "3d" },
        ],
      },
      {
        query: "log lines mentioning timeout",
        expected: [{ field: "message", operator: "contains", value: "timeout" }],
      },
      {
        query: "debug lines from fra1",
        expected: [
          { field: "severity", operator: "is", value: "DEBUG" },
          { field: "region", operator: "is", value: "fra1" },
        ],
      },
      {
        query: "everything in the last 30m",
        expected: [{ field: "since", operator: "is", value: "30m" }],
      },
    ],
  },
  {
    name: "audit",
    spec: auditSearchSpec,
    queries: [
      {
        query: "key deletions in the last 24h",
        expected: [
          { field: "events", operator: "is", value: "key.delete" },
          { field: "since", operator: "is", value: "24h" },
        ],
      },
      {
        query: "who created a role last week",
        expected: [
          { field: "events", operator: "is", value: "role.create" },
          { field: "since", operator: "is", value: "7d" },
        ],
      },
      {
        query: "deployment rollbacks",
        expected: [{ field: "events", operator: "is", value: "deployment.rollback" }],
      },
      {
        query: "anything that happened to webhooks in the past 3 days",
        expected: [
          { field: "events", operator: "is", value: "webhook.create" },
          { field: "events", operator: "is", value: "webhook.update" },
          { field: "events", operator: "is", value: "webhook.delete" },
          { field: "since", operator: "is", value: "3d" },
        ],
      },
      {
        query: "permission updates in the last 12h",
        expected: [
          { field: "events", operator: "is", value: "permission.update" },
          { field: "since", operator: "is", value: "12h" },
        ],
      },
    ],
  },
  {
    name: "ratelimit",
    spec: ratelimitSearchSpec,
    queries: [
      {
        query: "blocked requests in the last 2h",
        expected: [
          { field: "status", operator: "is", value: "blocked" },
          { field: "since", operator: "is", value: "2h" },
        ],
      },
      {
        query: "requests that were let through today",
        expected: [
          { field: "status", operator: "is", value: "passed" },
          { field: "since", operator: "is", value: "24h" },
        ],
      },
      {
        query: "everything for user_abc123def in the last 7 days",
        expected: [
          { field: "identifiers", operator: "is", value: "user_abc123def" },
          { field: "since", operator: "is", value: "7d" },
        ],
      },
      {
        query: "identifiers containing acme",
        expected: [{ field: "identifiers", operator: "contains", value: "acme" }],
      },
      {
        query: "who got rate limited in the past 30m",
        expected: [
          { field: "status", operator: "is", value: "blocked" },
          { field: "since", operator: "is", value: "30m" },
        ],
      },
    ],
  },
  {
    name: "key verifications",
    spec: keysOverviewSearchSpec,
    queries: [
      {
        query: "rate limited verifications in the last 24h",
        expected: [
          { field: "outcomes", operator: "is", value: "RATE_LIMITED" },
          { field: "since", operator: "is", value: "24h" },
        ],
      },
      {
        query: "expired keys in the past 7 days",
        expected: [
          { field: "outcomes", operator: "is", value: "EXPIRED" },
          { field: "since", operator: "is", value: "7d" },
        ],
      },
      {
        query: "successful verifications",
        expected: [{ field: "outcomes", operator: "is", value: "VALID" }],
      },
      {
        query: "keys tagged internal",
        expected: [{ field: "tags", operator: "is", value: "internal" }],
      },
      {
        query: "verifications that ran out of allowance in the last 12h",
        expected: [
          { field: "outcomes", operator: "is", value: "USAGE_EXCEEDED" },
          { field: "since", operator: "is", value: "12h" },
        ],
      },
    ],
  },
  {
    name: "keys",
    spec: keysListSearchSpec,
    queries: [
      {
        query: "keys whose name contains staging",
        expected: [{ field: "names", operator: "contains", value: "staging" }],
      },
      {
        query: "keys tagged internal",
        expected: [{ field: "tags", operator: "is", value: "internal" }],
      },
      {
        query: "keys for identity user_9fj3ks01",
        expected: [{ field: "identities", operator: "is", value: "user_9fj3ks01" }],
      },
      {
        query: "keys whose name starts with prod",
        expected: [{ field: "names", operator: "startsWith", value: "prod" }],
      },
    ],
  },
  {
    name: "roles",
    spec: rolesSearchSpec,
    queries: [
      {
        query: "roles whose name contains admin",
        expected: [{ field: "name", operator: "contains", value: "admin" }],
      },
      {
        query: "roles whose description mentions billing",
        expected: [{ field: "description", operator: "contains", value: "billing" }],
      },
      {
        query: "roles that grant a permission starting with api",
        expected: [{ field: "permissionSlug", operator: "startsWith", value: "api" }],
      },
      {
        query: "roles attached to a key named ci",
        expected: [{ field: "keyName", operator: "contains", value: "ci" }],
      },
    ],
  },
  {
    name: "permissions",
    spec: permissionsSearchSpec,
    queries: [
      {
        query: "permissions whose slug starts with api",
        expected: [{ field: "slug", operator: "startsWith", value: "api" }],
      },
      {
        query: "permissions whose name contains read",
        expected: [{ field: "name", operator: "contains", value: "read" }],
      },
      {
        query: "permissions whose description mentions delete",
        expected: [{ field: "description", operator: "contains", value: "delete" }],
      },
      {
        query: "permissions granted by the admin role",
        expected: [{ field: "roleName", operator: "contains", value: "admin" }],
      },
    ],
  },
  {
    name: "root keys",
    spec: rootKeysSearchSpec,
    queries: [
      {
        query: "root keys whose name contains ci",
        expected: [{ field: "name", operator: "contains", value: "ci" }],
      },
      {
        query: "root keys carrying the api.read_api permission",
        expected: [{ field: "permission", operator: "contains", value: "api.read_api" }],
      },
      {
        query: "root keys starting with unkey_3ZZ",
        expected: [{ field: "start", operator: "contains", value: "unkey_3ZZ" }],
      },
    ],
  },
  {
    name: "deployments",
    spec: deploymentsSearchSpec,
    queries: [
      {
        query: "failed deployments in the last 24h",
        expected: [
          { field: "status", operator: "is", value: "failed" },
          { field: "since", operator: "is", value: "24h" },
        ],
      },
      {
        query: "deployments on main",
        expected: [{ field: "branch", operator: "is", value: "main" }],
      },
      {
        query: "production deployments that are still building",
        expected: [
          { field: "status", operator: "is", value: "building" },
          { field: "environment", operator: "is", value: "production" },
        ],
      },
      {
        query: "deployments waiting for approval",
        expected: [{ field: "status", operator: "is", value: "blocked" }],
      },
      {
        query: "everything from the past 7 days",
        expected: [{ field: "since", operator: "is", value: "7d" }],
      },
    ],
  },
];
