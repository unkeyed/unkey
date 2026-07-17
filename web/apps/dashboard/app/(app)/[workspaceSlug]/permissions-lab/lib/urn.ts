/**
 * TypeScript mirror of pkg/urn (grammar v1) and the permission layer from
 * pkg/rbac/urn.go. This is prototype-only code for the permissions lab; the
 * Go packages remain the source of truth.
 *
 * Grammar: unkey:v1:{workspace_id}:{resource_path}
 * Permission: unkey:v1:{workspace_id}:{resource_path}#{action}
 *
 * Wildcards: "*" matches exactly one path segment; a trailing "**" matches
 * the base path and all descendants. The action wildcard "*" is valid only on
 * the global resource "**".
 */

export const URN_PREFIX = "unkey";
export const URN_VERSION = "v1";

export interface V1 {
  workspaceID: string;
  resource: string;
}

export function formatV1(urn: V1): string {
  return `${URN_PREFIX}:${URN_VERSION}:${urn.workspaceID}:${urn.resource}`;
}

export type Result<T> = { ok: true; value: T } | { ok: false; error: string };

/**
 * Mirrors urn.ParseV1. Returns a reasoned error instead of throwing so UIs
 * can render inline validation.
 */
export function parseV1(value: string): Result<V1> {
  const parts = splitN(value, ":", 4);
  if (parts.length !== 4) {
    return { ok: false, error: "expected 4 colon-separated fields" };
  }
  if (parts[0] !== URN_PREFIX) {
    return { ok: false, error: `prefix must be "${URN_PREFIX}"` };
  }
  if (parts[1] !== URN_VERSION) {
    return { ok: false, error: `version must be "${URN_VERSION}"` };
  }
  const wsError = validateWorkspaceID(parts[2]);
  if (wsError) {
    return { ok: false, error: `invalid workspace id: ${wsError}` };
  }
  const pathError = validateResourcePath(parts[3]);
  if (pathError) {
    return { ok: false, error: `invalid resource path: ${pathError}` };
  }
  return { ok: true, value: { workspaceID: parts[2], resource: parts[3] } };
}

/** Mirrors strings.SplitN: at most n parts, last part keeps remaining separators. */
function splitN(value: string, sep: string, n: number): string[] {
  const parts: string[] = [];
  let rest = value;
  while (parts.length < n - 1) {
    const idx = rest.indexOf(sep);
    if (idx === -1) {
      break;
    }
    parts.push(rest.slice(0, idx));
    rest = rest.slice(idx + sep.length);
  }
  parts.push(rest);
  return parts;
}

export function validateWorkspaceID(value: string): string | null {
  if (value === "") {
    return "must not be empty";
  }
  if (/[:#/]/.test(value)) {
    return 'must not contain ":", "#", or "/"';
  }
  return null;
}

/**
 * Mirrors validateResourcePath. Five invariants: no empty segments, no ":" or
 * "#", "*" only as a whole segment, "**" only as the final segment, and once
 * an id selector is "*" later selectors must also be "*" (or a trailing "**").
 */
export function validateResourcePath(path: string): string | null {
  const segments = path.split("/");
  let seenWildcardSelector = false;
  for (let i = 0; i < segments.length; i++) {
    const segment = segments[i];
    const isLastSegment = i === segments.length - 1;

    if (segment === "") {
      return "must not contain empty segments";
    }
    if (/[:#]/.test(segment)) {
      return 'must not contain ":" or "#"';
    }
    if (segment === "*") {
      seenWildcardSelector = true;
      continue;
    }
    if (segment === "**") {
      if (!isLastSegment) {
        return '"**" must be the last segment';
      }
      continue;
    }
    if (segment.includes("*")) {
      return '"*" must be a whole segment';
    }
    if (seenWildcardSelector) {
      if (isLastSegment || (segments[i + 1] !== "*" && segments[i + 1] !== "**")) {
        return 'specific selectors must not appear below "*"';
      }
    }
  }
  return null;
}

/**
 * Mirrors urn.V1.Covers: does `pattern` (a resource-name pattern) cover the
 * concrete `target`?
 */
export function covers(pattern: V1, target: V1): boolean {
  if (pattern.workspaceID !== target.workspaceID) {
    return false;
  }
  return pathCovers(pattern.resource, target.resource);
}

/** Segment-aware path matching, workspace already checked. */
export function pathCovers(pattern: string, target: string): boolean {
  let patternSegments = pattern.split("/");
  const targetSegments = target.split("/");

  if (patternSegments.length > 0 && patternSegments[patternSegments.length - 1] === "**") {
    patternSegments = patternSegments.slice(0, -1);
    return (
      targetSegments.length >= patternSegments.length &&
      segmentsMatch(patternSegments, targetSegments.slice(0, patternSegments.length))
    );
  }

  return (
    patternSegments.length === targetSegments.length &&
    segmentsMatch(patternSegments, targetSegments)
  );
}

function segmentsMatch(pattern: string[], target: string[]): boolean {
  for (let i = 0; i < pattern.length; i++) {
    if (pattern[i] !== "*" && pattern[i] !== target[i]) {
      return false;
    }
  }
  return true;
}

export function isPattern(resourcePath: string): boolean {
  return resourcePath.split("/").some((s) => s === "*" || s === "**");
}

// ---------------------------------------------------------------------------
// Permission layer (mirrors pkg/rbac/urn.go)
// ---------------------------------------------------------------------------

export interface Permission {
  urn: V1;
  action: string;
}

export function formatPermission(p: Permission): string {
  return `${formatV1(p.urn)}#${p.action}`;
}

const ACTION_RE = /^[a-z]+(_[a-z]+)*$/;

export function validateAction(action: string, resourcePath: string): string | null {
  if (action === "") {
    return "action must not be empty";
  }
  if (action === "*") {
    if (resourcePath !== "**") {
      return 'the action wildcard "*" is valid only on the global resource "**"';
    }
    return null;
  }
  if (!ACTION_RE.test(action)) {
    return "actions use lowercase words separated by underscores";
  }
  return null;
}

export function parsePermission(value: string): Result<Permission> {
  const hashIdx = value.lastIndexOf("#");
  if (hashIdx === -1) {
    return { ok: false, error: 'missing "#action" suffix' };
  }
  const urnPart = value.slice(0, hashIdx);
  const action = value.slice(hashIdx + 1);
  const parsed = parseV1(urnPart);
  if (!parsed.ok) {
    return parsed;
  }
  const actionError = validateAction(action, parsed.value.resource);
  if (actionError) {
    return { ok: false, error: actionError };
  }
  return { ok: true, value: { urn: parsed.value, action } };
}

/**
 * Mirrors permissionCovers: a grant matches a request when version, workspace,
 * resource pattern, and action all match. The request must be concrete.
 */
export function permissionCovers(grant: Permission, request: Permission): boolean {
  if (grant.action !== request.action) {
    if (!(grant.action === "*" && grant.urn.resource === "**")) {
      return false;
    }
  }
  return covers(grant.urn, request.urn);
}

/**
 * Diagnostic matching for the access debugger: explains why a grant does or
 * does not cover a request, so near-misses can be surfaced.
 */
export type MatchVerdict =
  | { kind: "match" }
  | { kind: "wrong_workspace" }
  | { kind: "wrong_action"; grantAction: string }
  | { kind: "resource_mismatch" }
  | { kind: "wildcard_too_shallow"; hint: string };

export function explainMatch(grant: Permission, request: Permission): MatchVerdict {
  if (grant.urn.workspaceID !== request.urn.workspaceID) {
    return { kind: "wrong_workspace" };
  }
  const resourceMatches = pathCovers(grant.urn.resource, request.urn.resource);
  const actionMatches =
    grant.action === request.action || (grant.action === "*" && grant.urn.resource === "**");

  if (resourceMatches && actionMatches) {
    return { kind: "match" };
  }
  if (resourceMatches) {
    return { kind: "wrong_action", grantAction: grant.action };
  }
  // Near-miss: pattern is a strict prefix of the target, so a descendant scope
  // would have reached it. Example: keyspaces/* does not reach
  // keyspaces/ks_1/keys/k_1, but keyspaces/** or keyspaces/*/keys/* would.
  const grantSegments = grant.urn.resource.split("/");
  const requestSegments = request.urn.resource.split("/");
  const base =
    grantSegments[grantSegments.length - 1] === "**" ? grantSegments.slice(0, -1) : grantSegments;
  if (
    base.length < requestSegments.length &&
    segmentsMatch(base, requestSegments.slice(0, base.length))
  ) {
    const descendant = [...base, "**"].join("/");
    return {
      kind: "wildcard_too_shallow",
      hint: `the pattern matches an ancestor of the requested resource; "${descendant}" would cover it`,
    };
  }
  return { kind: "resource_mismatch" };
}
