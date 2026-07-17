/**
 * Deterministic mock workspace for the permissions lab. Everything is
 * client-side; no tRPC or database access. The data is shaped so wildcard
 * grants have interesting coverage: multiple keyspaces with many keys, a
 * nested deploy tree, ratelimit namespaces with overrides.
 */

import { RESOURCE_TYPES, resourceTypeOfPath } from "./catalog";
import { formatPermission, formatV1, pathCovers } from "./urn";

export const WORKSPACE_ID = "ws_acme";
export const WORKSPACE_NAME = "ACME Inc";

export interface ConcreteResource {
  /** resource path, e.g. "keyspaces/ks_payments_prod/keys/key_pay_001" */
  path: string;
  /** catalog type, e.g. "key" */
  type: string;
  /** human name, e.g. "checkout-service" */
  label: string;
}

export interface MockKeyspace {
  id: string;
  name: string;
  keys: { id: string; name: string }[];
}

export const KEYSPACES: MockKeyspace[] = [
  {
    id: "ks_payments_prod",
    name: "payments-prod",
    keys: numbered(18, (i) => ({
      id: `key_pay_${pad(i)}`,
      name: `payments-prod key ${pad(i)}`,
    })),
  },
  {
    id: "ks_payments_staging",
    name: "payments-staging",
    keys: numbered(7, (i) => ({
      id: `key_stg_${pad(i)}`,
      name: `payments-staging key ${pad(i)}`,
    })),
  },
  {
    id: "ks_internal_tools",
    name: "internal-tools",
    keys: numbered(4, (i) => ({
      id: `key_int_${pad(i)}`,
      name: `internal-tools key ${pad(i)}`,
    })),
  },
];

export interface MockIdentity {
  id: string;
  externalID: string;
}

export const IDENTITIES: MockIdentity[] = numbered(12, (i) => ({
  id: `id_cust_${pad(i)}`,
  externalID: `customer_${pad(i)}@acme.com`,
}));

export interface MockRatelimitNamespace {
  id: string;
  name: string;
  overrides: { id: string; identifier: string }[];
}

export const RATELIMIT_NAMESPACES: MockRatelimitNamespace[] = [
  {
    id: "rlns_api_global",
    name: "api.global",
    overrides: [
      { id: "rlor_vip_01", identifier: "customer_003@acme.com" },
      { id: "rlor_vip_02", identifier: "customer_007@acme.com" },
    ],
  },
  {
    id: "rlns_checkout",
    name: "checkout",
    overrides: [{ id: "rlor_burst_01", identifier: "customer_001@acme.com" }],
  },
];

export interface MockEnvironment {
  id: string;
  name: string;
  deployments: { id: string; name: string }[];
  domains: { id: string; name: string }[];
  variables: { id: string; name: string }[];
}

export interface MockApp {
  id: string;
  name: string;
  environments: MockEnvironment[];
}

export interface MockProject {
  id: string;
  name: string;
  apps: MockApp[];
}

export const PROJECTS: MockProject[] = [
  {
    id: "proj_storefront",
    name: "storefront",
    apps: [
      {
        id: "app_web",
        name: "web",
        environments: [
          {
            id: "env_production",
            name: "production",
            deployments: [
              { id: "dep_9f3a21", name: "dep_9f3a21" },
              { id: "dep_8c1b44", name: "dep_8c1b44" },
              { id: "dep_7d0e19", name: "dep_7d0e19" },
            ],
            domains: [{ id: "dom_acme_com", name: "acme.com" }],
            variables: [
              { id: "var_stripe_key", name: "STRIPE_KEY" },
              { id: "var_log_level", name: "LOG_LEVEL" },
            ],
          },
          {
            id: "env_preview",
            name: "preview",
            deployments: [{ id: "dep_pre_112", name: "dep_pre_112" }],
            domains: [{ id: "dom_preview", name: "preview.acme.com" }],
            variables: [{ id: "var_log_level_pre", name: "LOG_LEVEL" }],
          },
        ],
      },
      {
        id: "app_checkout",
        name: "checkout",
        environments: [
          {
            id: "env_production",
            name: "production",
            deployments: [{ id: "dep_chk_551", name: "dep_chk_551" }],
            domains: [{ id: "dom_checkout", name: "checkout.acme.com" }],
            variables: [{ id: "var_psp_secret", name: "PSP_SECRET" }],
          },
        ],
      },
    ],
  },
  {
    id: "proj_data_pipeline",
    name: "data-pipeline",
    apps: [
      {
        id: "app_ingest",
        name: "ingest",
        environments: [
          {
            id: "env_production",
            name: "production",
            deployments: [{ id: "dep_ing_204", name: "dep_ing_204" }],
            domains: [],
            variables: [{ id: "var_bucket", name: "BUCKET" }],
          },
        ],
      },
    ],
  },
];

export interface MockRole {
  id: string;
  name: string;
  description: string;
  /** canonical permission strings */
  permissions: string[];
}

export const ROLES: MockRole[] = [
  {
    id: "role_admin",
    name: "admin",
    description: "Full access to everything in the workspace",
    permissions: [perm("**", "*")],
  },
  {
    id: "role_key_minter",
    name: "key-minter",
    description: "Creates and manages keys in the payments keyspaces",
    permissions: [
      perm("keyspaces/ks_payments_prod", "create_key"),
      perm("keyspaces/ks_payments_staging", "create_key"),
      perm("keyspaces/*/keys/*", "read_key"),
      perm("keyspaces/*/keys/*", "update_key"),
    ],
  },
  {
    id: "role_observer",
    name: "observer",
    description: "Read-only across keyspaces and identities",
    permissions: [
      perm("keyspaces/*", "read_keyspace"),
      perm("keyspaces/*/keys/*", "read_key"),
      perm("identities/*", "read_identity"),
    ],
  },
  {
    id: "role_deployer",
    name: "deployer",
    description: "Ships deployments for the storefront project",
    permissions: [
      perm("projects/proj_storefront/**", "read_deployment"),
      perm("projects/proj_storefront/apps/*/environments/*", "create_deployment"),
      perm("projects/proj_storefront/**", "delete_deployment"),
    ],
  },
];

export interface MockPrincipal {
  id: string;
  name: string;
  kind: "root_key";
  /** direct permission strings, canonical form */
  permissions: string[];
  /** attached role ids */
  roles: string[];
  createdAt: string;
}

export const PRINCIPALS: MockPrincipal[] = [
  {
    id: "unkey_root_ci",
    name: "CI deploy key",
    kind: "root_key",
    permissions: [],
    roles: ["role_deployer"],
    createdAt: "2026-03-02",
  },
  {
    id: "unkey_root_payments",
    name: "Payments service",
    kind: "root_key",
    permissions: [
      perm("keyspaces/ks_payments_prod", "create_key"),
      perm("keyspaces/ks_payments_prod/keys/*", "verify_key"),
      perm("keyspaces/ks_payments_prod/keys/*", "read_key"),
    ],
    roles: [],
    createdAt: "2026-01-15",
  },
  {
    id: "unkey_root_support",
    name: "Support dashboard",
    kind: "root_key",
    permissions: [
      perm("identities/*", "read_identity"),
      perm("ratelimits/namespaces/rlns_api_global", "set_override"),
    ],
    roles: ["role_observer"],
    createdAt: "2026-02-20",
  },
  {
    id: "unkey_root_admin",
    name: "Break-glass admin",
    kind: "root_key",
    permissions: [],
    roles: ["role_admin"],
    createdAt: "2025-11-30",
  },
  {
    id: "unkey_root_analytics",
    name: "Analytics exporter",
    kind: "root_key",
    permissions: [perm("keyspaces/*", "read_keyspace"), perm("keyspaces/*/keys/*", "read_key")],
    roles: [],
    createdAt: "2026-05-11",
  },
];

/** Legacy tuple permissions still present on some principals (exact-string match only). */
export const LEGACY_PERMISSIONS: Record<string, string[]> = {
  unkey_root_support: ["api.ks_internal_tools.read_key"],
};

// ---------------------------------------------------------------------------
// Derived data
// ---------------------------------------------------------------------------

/** Every concrete resource in the mock workspace, used for coverage counts. */
export const ALL_RESOURCES: ConcreteResource[] = buildAllResources();

function buildAllResources(): ConcreteResource[] {
  const out: ConcreteResource[] = [];

  for (const ks of KEYSPACES) {
    out.push({ path: `keyspaces/${ks.id}`, type: "keyspace", label: ks.name });
    for (const key of ks.keys) {
      out.push({
        path: `keyspaces/${ks.id}/keys/${key.id}`,
        type: "key",
        label: key.name,
      });
    }
  }

  for (const identity of IDENTITIES) {
    out.push({ path: `identities/${identity.id}`, type: "identity", label: identity.externalID });
  }

  for (const ns of RATELIMIT_NAMESPACES) {
    out.push({
      path: `ratelimits/namespaces/${ns.id}`,
      type: "ratelimit_namespace",
      label: ns.name,
    });
    for (const o of ns.overrides) {
      out.push({
        path: `ratelimits/namespaces/${ns.id}/overrides/${o.id}`,
        type: "ratelimit_override",
        label: o.identifier,
      });
    }
  }

  for (const role of ROLES) {
    out.push({ path: `rbac/roles/${role.id}`, type: "role", label: role.name });
  }

  for (const project of PROJECTS) {
    out.push({ path: `projects/${project.id}`, type: "project", label: project.name });
    for (const app of project.apps) {
      const appPath = `projects/${project.id}/apps/${app.id}`;
      out.push({ path: appPath, type: "app", label: app.name });
      for (const env of app.environments) {
        const envPath = `${appPath}/environments/${env.id}`;
        out.push({ path: envPath, type: "environment", label: env.name });
        for (const dep of env.deployments) {
          out.push({
            path: `${envPath}/deployments/${dep.id}`,
            type: "deployment",
            label: dep.name,
          });
        }
        for (const dom of env.domains) {
          out.push({ path: `${envPath}/domains/${dom.id}`, type: "domain", label: dom.name });
        }
        for (const v of env.variables) {
          out.push({ path: `${envPath}/variables/${v.id}`, type: "variable", label: v.name });
        }
      }
    }
  }

  return out;
}

/** Concrete resources covered by a resource-path pattern. */
export function coverage(pattern: string): ConcreteResource[] {
  return ALL_RESOURCES.filter((r) => pathCovers(pattern, r.path));
}

/** Human label for a concrete path, falling back to the last segment. */
export function labelForPath(path: string): string {
  const hit = ALL_RESOURCES.find((r) => r.path === path);
  if (hit) {
    return hit.label;
  }
  const segments = path.split("/");
  return segments[segments.length - 1];
}

export interface SegmentSuggestion {
  /** the segment value to insert, e.g. "ks_payments_prod" or "*" */
  value: string;
  /** display label, e.g. "payments-prod" */
  label: string;
  kind: "collection" | "id" | "wildcard" | "descendants";
}

/**
 * Segment-level autocomplete: given the segments typed so far, suggest what
 * can come next. Derived from ALL_RESOURCES plus the wildcard operators the
 * grammar allows at that position.
 */
export function suggestNextSegments(prefixSegments: string[]): SegmentSuggestion[] {
  const depth = prefixSegments.length;
  const seen = new Map<string, SegmentSuggestion>();

  for (const resource of ALL_RESOURCES) {
    const segments = resource.path.split("/");
    if (segments.length <= depth) {
      continue;
    }
    let matchesPrefix = true;
    for (let i = 0; i < depth; i++) {
      if (prefixSegments[i] !== "*" && prefixSegments[i] !== segments[i]) {
        matchesPrefix = false;
        break;
      }
    }
    if (!matchesPrefix) {
      continue;
    }
    const next = segments[depth];
    if (seen.has(next)) {
      continue;
    }
    const isID =
      segments.length === depth + 1 ||
      resourceTypeOfPath(segments.slice(0, depth + 1).join("/")) !== null;
    seen.set(next, {
      value: next,
      label: segments.length === depth + 1 ? resource.label : next,
      kind: isID && !isCollectionLabel(next) ? "id" : "collection",
    });
  }

  const suggestions = [...seen.values()];

  // Wildcards: "*" replaces exactly one segment; "**" is valid as the final
  // segment anywhere, including the bare global pattern.
  if (depth > 0 || suggestions.length > 0) {
    if (suggestions.some((s) => s.kind === "id")) {
      suggestions.push({ value: "*", label: "any (one segment)", kind: "wildcard" });
    }
    suggestions.push({ value: "**", label: "this and all descendants", kind: "descendants" });
  }

  return suggestions;
}

const COLLECTION_LABELS = new Set(
  RESOURCE_TYPES.flatMap((t) => t.shape.filter((s) => !s.startsWith("{"))),
);

function isCollectionLabel(segment: string): boolean {
  return COLLECTION_LABELS.has(segment);
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

export function perm(resourcePath: string, action: string): string {
  return formatPermission({
    urn: { workspaceID: WORKSPACE_ID, resource: resourcePath },
    action,
  });
}

export function urnString(resourcePath: string): string {
  return formatV1({ workspaceID: WORKSPACE_ID, resource: resourcePath });
}

function numbered<T>(count: number, build: (i: number) => T): T[] {
  return Array.from({ length: count }, (_, idx) => build(idx + 1));
}

function pad(n: number): string {
  return String(n).padStart(3, "0");
}
