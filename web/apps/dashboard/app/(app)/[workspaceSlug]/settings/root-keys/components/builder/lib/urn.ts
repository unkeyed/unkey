import type { UnkeyUrnPermission } from "@unkey/rbac";
import { CATALOGUES, catalogueRows } from "./catalogue";
import { type Action, type PermissionRow, instancePath, resolveInstance } from "./catalogue.types";
import { type Policy, rowActions } from "./policy";

export type RowGrant = {
  path: string;
  action: string;
};

export function buildUrn(workspaceId: string, path: string, action: string): UnkeyUrnPermission {
  return `unkey:v1:${workspaceId}:${path}#${action}`;
}

// `instance` is the path the policy stands on, already resolved: a concrete
// instance path, or what the scope calls every instance.
export function rowActionGrants(row: PermissionRow, action: Action, instance: string): RowGrant[] {
  return row.actions[action].map((grant) => ({
    path: instancePath(grant.path, instance),
    action: grant.name,
  }));
}

export function rowActionUrns(
  workspaceId: string,
  row: PermissionRow,
  action: Action,
  instance: string,
): UnkeyUrnPermission[] {
  return rowActionGrants(row, action, instance).map((grant) =>
    buildUrn(workspaceId, grant.path, grant.action),
  );
}

export function buildUrns(workspaceId: string, policies: readonly Policy[]): UnkeyUrnPermission[] {
  const urns = new Set<UnkeyUrnPermission>();
  for (const policy of policies) {
    const catalogue = CATALOGUES[policy.scope];
    for (const row of catalogueRows(catalogue)) {
      for (const action of rowActions(policy.selection, row.id)) {
        for (const instance of policy.instances) {
          for (const urn of rowActionUrns(
            workspaceId,
            row,
            action,
            resolveInstance(catalogue, instance),
          )) {
            urns.add(urn);
          }
        }
      }
    }
  }
  return [...urns];
}
