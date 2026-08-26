import type { UnkeyUrnPermission } from "@unkey/rbac";
import { CATALOGUES, catalogueRows } from "./catalogue";
import { ACTIONS, type Action, type PermissionRow, instancePath } from "./catalogue.types";
import { type Policy, rowActions } from "./policy";

export type RowGrant = {
  path: string;
  action: string;
};

export function buildUrn(workspaceId: string, path: string, action: string): UnkeyUrnPermission {
  return `unkey:v1:${workspaceId}:${path}#${action}`;
}

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
    for (const row of catalogueRows(CATALOGUES[policy.scope])) {
      for (const action of rowActions(policy.selection, row.id)) {
        for (const instance of policy.instances) {
          for (const urn of rowActionUrns(workspaceId, row, action, instance)) {
            urns.add(urn);
          }
        }
      }
    }
  }
  return [...urns];
}

export function rowGrants(row: PermissionRow, instances: readonly string[]): RowGrant[] {
  const grants = new Map<string, RowGrant>();
  for (const action of ACTIONS) {
    for (const instance of instances) {
      for (const grant of rowActionGrants(row, action, instance)) {
        grants.set(`${grant.path}#${grant.action}`, grant);
      }
    }
  }
  return [...grants.values()];
}
