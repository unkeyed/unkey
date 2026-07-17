/**
 * Pure access computation for the share sheet: given the lab principals and
 * roles, find every grant whose resource pattern covers the selected concrete
 * resource, and remember how the principal got it (direct vs role) so the UI
 * can be honest about what is revocable here and what is managed elsewhere.
 */

import type { MockPrincipal, MockRole } from "../lib/mock-data";
import { WORKSPACE_ID } from "../lib/mock-data";
import { covers, parsePermission } from "../lib/urn";

export interface MatchedGrant {
  /** canonical permission string, e.g. "unkey:v1:ws_acme:keyspaces/*#read_keyspace" */
  permission: string;
  action: string;
  /** the grant's resource pattern (may be wider than the selected resource) */
  resourcePattern: string;
  /** true when the grant pattern is not the concrete resource path itself */
  inherited: boolean;
  /** true when the grant string sits directly on the principal */
  direct: boolean;
  /** roles that carry this grant */
  viaRoles: MockRole[];
}

export interface AccessRow {
  principal: MockPrincipal;
  grants: MatchedGrant[];
}

/** A grant is revocable from the share sheet only when it is a direct grant on exactly this resource. */
export function isRevocableHere(grant: MatchedGrant): boolean {
  return grant.direct && !grant.inherited;
}

export function accessRowsForResource(
  principals: MockPrincipal[],
  roles: MockRole[],
  resourcePath: string,
): AccessRow[] {
  const rolesByID = new Map(roles.map((role) => [role.id, role]));
  const target = { workspaceID: WORKSPACE_ID, resource: resourcePath };
  const rows: AccessRow[] = [];

  for (const principal of principals) {
    const byPermission = new Map<string, MatchedGrant>();

    const consider = (permission: string, source: { direct: boolean; role?: MockRole }) => {
      const existing = byPermission.get(permission);
      if (existing) {
        if (source.direct) {
          existing.direct = true;
        }
        if (source.role) {
          const role = source.role;
          if (!existing.viaRoles.some((r) => r.id === role.id)) {
            existing.viaRoles.push(role);
          }
        }
        return;
      }
      const parsed = parsePermission(permission);
      if (!parsed.ok || !covers(parsed.value.urn, target)) {
        return;
      }
      byPermission.set(permission, {
        permission,
        action: parsed.value.action,
        resourcePattern: parsed.value.urn.resource,
        inherited: parsed.value.urn.resource !== resourcePath,
        direct: source.direct,
        viaRoles: source.role ? [source.role] : [],
      });
    };

    for (const permission of principal.permissions) {
      consider(permission, { direct: true });
    }
    for (const roleID of principal.roles) {
      const role = rolesByID.get(roleID);
      if (!role) {
        continue;
      }
      for (const permission of role.permissions) {
        consider(permission, { direct: false, role });
      }
    }

    const grants = [...byPermission.values()].sort((a, b) => a.action.localeCompare(b.action));
    if (grants.length > 0) {
      rows.push({ principal, grants });
    }
  }

  return rows;
}
