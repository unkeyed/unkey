import {
  ALL_INSTANCES,
  type Action,
  type PermissionRow,
  RESOURCE_SCOPES,
  type ResourceScope,
  type ScopeCatalogue,
  offeredActions,
  resolveInstance,
} from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/builder/lib/catalogue.types";
import { rowActionUrns } from "@/app/(app)/[workspaceSlug]/settings/root-keys/components/builder/lib/urn";
import { parseUrnPermissionParts } from "@unkey/rbac";

export function uncoveredPermissions(
  granted: readonly string[],
  requested: readonly string[],
): string[] {
  const grants = granted.flatMap((grant) => {
    const parts = parseUrnPermissionParts(grant);
    return parts ? [parts] : [];
  });
  return requested.filter((permission) => {
    const target = parseUrnPermissionParts(permission);
    return (
      target === null ||
      !grants.some(
        (grant) =>
          grant.workspaceId === target.workspaceId &&
          (grant.action === "*" || grant.action === target.action) &&
          resourceCovers(grant.resourcePath, target.resourcePath),
      )
    );
  });
}

function resourceCovers(pattern: string, target: string): boolean {
  const patternSegments = pattern.split("/");
  const targetSegments = target.split("/");
  if (patternSegments.at(-1) === "**") {
    const base = patternSegments.slice(0, -1);
    return (
      targetSegments.length >= base.length &&
      segmentsMatch(base, targetSegments.slice(0, base.length))
    );
  }
  return (
    patternSegments.length === targetSegments.length &&
    segmentsMatch(patternSegments, targetSegments)
  );
}

function segmentsMatch(pattern: readonly string[], target: readonly string[]): boolean {
  return pattern.every((segment, index) => segment === "*" || segment === target[index]);
}

export function grantableCatalogues(
  catalogues: Record<ResourceScope, ScopeCatalogue>,
  workspaceId: string,
  grantable: readonly string[],
): Record<ResourceScope, ScopeCatalogue> {
  const restricted = { ...catalogues };
  for (const scope of RESOURCE_SCOPES) {
    restricted[scope] = grantableCatalogue(catalogues[scope], workspaceId, grantable);
  }
  return restricted;
}

function grantableCatalogue(
  catalogue: ScopeCatalogue,
  workspaceId: string,
  grantable: readonly string[],
): ScopeCatalogue {
  const instance = resolveInstance(catalogue, ALL_INSTANCES);
  const groups = catalogue.groups.flatMap((group) => {
    const rows = group.rows.flatMap((row) => {
      const kept = grantableRow(
        row,
        (action) =>
          uncoveredPermissions(grantable, rowActionUrns(workspaceId, row, action, instance))
            .length === 0,
      );
      return offeredActions(kept).length > 0 ? [kept] : [];
    });
    return rows.length > 0 ? [{ ...group, rows }] : [];
  });
  return { ...catalogue, groups };
}

function grantableRow(row: PermissionRow, covered: (action: Action) => boolean): PermissionRow {
  const keep = (action: Action) => (covered(action) ? row.actions[action] : []);
  return {
    ...row,
    actions: {
      read: keep("read"),
      write: keep("write"),
      delete: keep("delete"),
      verify: keep("verify"),
      decrypt: keep("decrypt"),
      limit: keep("limit"),
    },
  };
}
