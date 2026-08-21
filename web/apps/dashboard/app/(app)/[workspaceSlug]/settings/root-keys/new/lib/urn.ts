import { catalogueFor, catalogueRows } from "./catalogue";
import type { Action } from "./catalogue.types";
import { type Policy, rowActions } from "./policy";

export function urnActions(resource: string, action: Action): string[] {
  switch (action) {
    case "read":
      return [`read_${resource}`];
    case "write":
      return [`create_${resource}`, `update_${resource}`];
    case "delete":
      return [`delete_${resource}`];
  }
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
      for (const action of rowActions(policy.selection, row.id)) {
        for (const name of urnActions(row.resource, action)) {
          urns.add(buildUrn(workspaceId, row.path, name));
        }
      }
    }
  }
  return [...urns];
}
