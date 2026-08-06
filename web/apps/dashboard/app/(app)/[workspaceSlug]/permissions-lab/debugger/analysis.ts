/**
 * Pure analysis for the access debugger: classify every effective grant of a
 * principal against one concrete request, so the UI can show the exact grant
 * that matched, or the closest near-misses with a suggested fix.
 */

import type { MockPrincipal, MockRole } from "../lib/mock-data";
import { WORKSPACE_ID, perm } from "../lib/mock-data";
import { type Permission, explainMatch, parsePermission } from "../lib/urn";

export interface DebuggerQuery {
  principalID: string;
  /** concrete resource path, e.g. "keyspaces/ks_payments_prod/keys/key_pay_001" */
  resourcePath: string;
  action: string;
}

export interface AnalyzedGrant {
  /** canonical permission string */
  permission: string;
  /** "direct" and/or "via role {name}" */
  sources: string[];
}

export interface WrongActionGrant extends AnalyzedGrant {
  /** the action this grant does allow on the requested resource */
  grantAction: string;
}

export interface ShallowGrant extends AnalyzedGrant {
  /** explanation from explainMatch */
  hint: string;
  /** the grant's base path widened to cover all descendants */
  suggestedPath: string;
  /** canonical permission string that would cover the request */
  suggestedPermission: string;
}

export interface Analysis {
  allowed: boolean;
  /** canonical string of the requested permission */
  requestString: string;
  /** grants that cover the request */
  matched: AnalyzedGrant[];
  /** every grant that did not match, in original order */
  others: AnalyzedGrant[];
  /** right resource, wrong action */
  wrongAction: WrongActionGrant[];
  /** pattern stops above the requested resource */
  tooShallow: ShallowGrant[];
  /** different resource entirely (or a different workspace) */
  unrelated: AnalyzedGrant[];
}

export function analyzeAccess(
  principal: MockPrincipal,
  roles: MockRole[],
  effective: string[],
  query: DebuggerQuery,
): Analysis {
  const request: Permission = {
    urn: { workspaceID: WORKSPACE_ID, resource: query.resourcePath },
    action: query.action,
  };

  const matched: AnalyzedGrant[] = [];
  const others: AnalyzedGrant[] = [];
  const wrongAction: WrongActionGrant[] = [];
  const tooShallow: ShallowGrant[] = [];
  const unrelated: AnalyzedGrant[] = [];

  for (const permission of effective) {
    const grant: AnalyzedGrant = {
      permission,
      sources: sourcesOf(permission, principal, roles),
    };
    const parsed = parsePermission(permission);
    if (!parsed.ok) {
      // Store data is canonical so this should not happen; surface the grant
      // as unrelated instead of dropping it silently.
      others.push(grant);
      unrelated.push(grant);
      continue;
    }
    const verdict = explainMatch(parsed.value, request);
    if (verdict.kind === "match") {
      matched.push(grant);
      continue;
    }
    others.push(grant);
    switch (verdict.kind) {
      case "wrong_action":
        wrongAction.push({ ...grant, grantAction: verdict.grantAction });
        break;
      case "wildcard_too_shallow": {
        const suggestedPath = descendantPattern(parsed.value.urn.resource);
        tooShallow.push({
          ...grant,
          hint: verdict.hint,
          suggestedPath,
          suggestedPermission: perm(suggestedPath, query.action),
        });
        break;
      }
      default:
        unrelated.push(grant);
    }
  }

  return {
    allowed: matched.length > 0,
    requestString: perm(query.resourcePath, query.action),
    matched,
    others,
    wrongAction,
    tooShallow,
    unrelated,
  };
}

/**
 * The grant's base path widened with a trailing "**" so it reaches every
 * descendant. Mirrors the construction inside explainMatch's hint.
 */
function descendantPattern(resource: string): string {
  const segments = resource.split("/");
  const base = segments[segments.length - 1] === "**" ? segments.slice(0, -1) : segments;
  return [...base, "**"].join("/");
}

function sourcesOf(permission: string, principal: MockPrincipal, roles: MockRole[]): string[] {
  const sources: string[] = [];
  if (principal.permissions.includes(permission)) {
    sources.push("direct");
  }
  for (const roleID of principal.roles) {
    const role = roles.find((r) => r.id === roleID);
    if (role?.permissions.includes(permission)) {
      sources.push(`via role ${role.name}`);
    }
  }
  return sources;
}

/** Human phrasing for a grant's provenance. */
export function sourceText(sources: string[]): string {
  if (sources.length === 0) {
    return "unknown source";
  }
  return sources.map((s) => (s === "direct" ? "direct grant" : s)).join(", ");
}
