/**
 * Derivation logic for the matrix concept. Grants are parsed once per
 * principal (buildGrantIndex) and every cell resolves against that index, so
 * the grid stays cheap even at hundreds of rows.
 */

import { RESOURCE_TYPES, type ResourceTypeDef, resourceTypesOfPath } from "../lib/catalog";
import {
  ALL_RESOURCES,
  type ConcreteResource,
  type MockPrincipal,
  type MockRole,
  WORKSPACE_ID,
} from "../lib/mock-data";
import {
  type Permission,
  isPattern,
  parsePermission,
  pathCovers,
  permissionCovers,
} from "../lib/urn";

export type GrantSource = { kind: "direct" } | { kind: "role"; roleID: string; roleName: string };

export interface ParsedGrant {
  /** canonical permission string as stored on the principal or role */
  raw: string;
  permission: Permission;
  /** resource path contains "*" or "**" */
  pattern: boolean;
  /** where the grant comes from; the same string can arrive via several paths */
  sources: GrantSource[];
}

/**
 * Parse every effective grant of a principal exactly once, keeping source
 * attribution (direct vs. role) so cells can explain themselves.
 */
export function buildGrantIndex(
  principal: MockPrincipal,
  rolesByID: Map<string, MockRole>,
): ParsedGrant[] {
  const byRaw = new Map<string, ParsedGrant>();

  const add = (raw: string, source: GrantSource) => {
    const existing = byRaw.get(raw);
    if (existing) {
      existing.sources.push(source);
      return;
    }
    const parsed = parsePermission(raw);
    if (!parsed.ok) {
      // legacy or malformed strings never light URN cells
      return;
    }
    byRaw.set(raw, {
      raw,
      permission: parsed.value,
      pattern: isPattern(parsed.value.urn.resource),
      sources: [source],
    });
  };

  for (const raw of principal.permissions) {
    add(raw, { kind: "direct" });
  }
  for (const roleID of principal.roles) {
    const role = rolesByID.get(roleID);
    if (!role) {
      continue;
    }
    for (const raw of role.permissions) {
      add(raw, { kind: "role", roleID, roleName: role.name });
    }
  }
  return [...byRaw.values()];
}

export type CellState =
  /** exact concrete grant held directly; revocable from the grid */
  | { kind: "direct"; grant: ParsedGrant }
  /** covered by a pattern grant or inherited through a role; explain, do not edit */
  | { kind: "covered"; grants: ParsedGrant[] }
  | { kind: "none" };

export function cellState(grants: ParsedGrant[], resourcePath: string, action: string): CellState {
  const request: Permission = {
    urn: { workspaceID: WORKSPACE_ID, resource: resourcePath },
    action,
  };
  const matching = grants.filter((g) => permissionCovers(g.permission, request));
  if (matching.length === 0) {
    return { kind: "none" };
  }
  const direct = matching.find(
    (g) =>
      g.permission.urn.resource === resourcePath &&
      g.permission.action === action &&
      g.sources.some((s) => s.kind === "direct"),
  );
  if (direct) {
    return { kind: "direct", grant: direct };
  }
  return { kind: "covered", grants: matching };
}

/** Human explanation for a covering grant, e.g. `via keyspaces/*#read_key (role: observer)`. */
export function sourceLabel(grant: ParsedGrant): string {
  const parts = grant.sources.map((s) => (s.kind === "direct" ? "direct" : `role: ${s.roleName}`));
  return `via ${grant.permission.urn.resource}#${grant.permission.action} (${parts.join(", ")})`;
}

// ---------------------------------------------------------------------------
// Families
// ---------------------------------------------------------------------------

export interface Family {
  type: string;
  /** plural section title */
  title: string;
  def: ResourceTypeDef;
  /** every concrete resource of this type in the mock workspace */
  resources: ConcreteResource[];
}

const FAMILY_ORDER: { type: string; title: string }[] = [
  { type: "keyspace", title: "Keyspaces" },
  { type: "key", title: "Keys" },
  { type: "identity", title: "Identities" },
  { type: "ratelimit_namespace", title: "Ratelimit namespaces" },
  { type: "ratelimit_override", title: "Ratelimit overrides" },
  { type: "project", title: "Projects" },
  { type: "app", title: "Apps" },
  { type: "environment", title: "Environments" },
  { type: "deployment", title: "Deployments" },
];

export const FAMILIES: Family[] = FAMILY_ORDER.flatMap(({ type, title }) => {
  const def = RESOURCE_TYPES.find((t) => t.type === type);
  if (!def) {
    return [];
  }
  return [{ type, title, def, resources: ALL_RESOURCES.filter((r) => r.type === type) }];
});

// ---------------------------------------------------------------------------
// Pattern rows
// ---------------------------------------------------------------------------

export interface PatternRow {
  grant: ParsedGrant;
  /** actions of this family the grant lights up */
  litActions: Set<string>;
  /** concrete resources of this family the pattern currently covers */
  covered: ConcreteResource[];
  /**
   * Replaceable with concrete grants from this section: the grant is held
   * directly and its action is specific (a wildcard action spans families).
   */
  materializable: boolean;
}

export function patternRowsForFamily(grants: ParsedGrant[], family: Family): PatternRow[] {
  const rows: PatternRow[] = [];
  for (const grant of grants) {
    if (!grant.pattern) {
      continue;
    }
    const resource = grant.permission.urn.resource;
    const wildcardAction = grant.permission.action === "*";
    const lit = family.def.actions.filter(
      (a) => wildcardAction || a.action === grant.permission.action,
    );
    if (lit.length === 0) {
      continue;
    }
    if (!resourceTypesOfPath(resource).some((t) => t.type === family.type)) {
      continue;
    }
    rows.push({
      grant,
      litActions: new Set(lit.map((a) => a.action)),
      covered: family.resources.filter((r) => pathCovers(resource, r.path)),
      materializable: !wildcardAction && grant.sources.some((s) => s.kind === "direct"),
    });
  }
  return rows;
}
