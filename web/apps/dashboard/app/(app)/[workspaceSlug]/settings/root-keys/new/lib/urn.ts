import { catalogueFor, catalogueRows } from "./catalogue";
import {
  ACTIONS,
  type Action,
  type ActionGrant,
  INSTANCE_TOKEN,
  type PermissionRow,
} from "./catalogue.types";
import { ALL_INSTANCES, type Policy, rowActions } from "./policy";

export function urnActions(row: PermissionRow, action: Action): ActionGrant[] {
  const declared = row.actions?.[action];
  if (declared) {
    return [...declared];
  }
  return [];
}

export function instancePath(path: string, instance: string, allPath?: string): string {
  if (instance === ALL_INSTANCES && allPath !== undefined) {
    return allPath;
  }
  return path.split(INSTANCE_TOKEN).join(instance === ALL_INSTANCES ? "*" : instance);
}

export function isValidResourcePath(path: string): boolean {
  if (path.length === 0) {
    return false;
  }
  const segments = path.split("/");
  return segments.every((segment, index) => {
    if (segment.length === 0) {
      return false;
    }
    if (segment === "**") {
      return index === segments.length - 1;
    }
    return !segment.includes("*") || segment === "*";
  });
}

export function buildUrn(workspaceId: string, path: string, action: string): string {
  return `unkey:v1:${workspaceId}:${path}#${action}`;
}

export function buildUrns(workspaceId: string, policies: readonly Policy[]): string[] {
  const urns = new Set<string>();
  for (const policy of policies) {
    for (const row of catalogueRows(catalogueFor(policy.scope))) {
      for (const action of rowActions(policy.selection, row.id, row)) {
        for (const grant of urnActions(row, action)) {
          for (const instance of policy.instances) {
            urns.add(
              buildUrn(
                workspaceId,
                instancePath(grant.path ?? row.path, instance, row.allPath),
                grant.name,
              ),
            );
          }
        }
      }
    }
  }
  return [...urns];
}

export function rowGrants(row: PermissionRow, instances: readonly string[]): string[] {
  const grants = new Set<string>();
  for (const action of ACTIONS) {
    for (const grant of urnActions(row, action)) {
      for (const instance of instances) {
        grants.add(`${instancePath(grant.path ?? row.path, instance, row.allPath)}#${grant.name}`);
      }
    }
  }
  return [...grants];
}

export function grantPaths(grants: readonly string[]): string[] {
  return [...new Set(grants.map((grant) => grant.split("#")[0]))];
}
