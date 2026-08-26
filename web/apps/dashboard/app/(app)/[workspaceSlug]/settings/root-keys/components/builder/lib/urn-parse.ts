import { parseUrnPermissionParts } from "@unkey/rbac";
import { CATALOGUES, catalogueRows } from "./catalogue";
import {
  ACTIONS,
  ALL_INSTANCES,
  INSTANCE_TOKEN,
  type PermissionRow,
  type PermissionSelection,
  RESOURCE_SCOPES,
  type ResourceScope,
  type ScopeCatalogue,
} from "./catalogue.types";
import { type Policy, rowActions, setRowActions } from "./policy";
import { rowActionUrns } from "./urn";

// Scopes claim grants in this order. Workspace is left out because it is every
// other scope wildcarded, so it would claim grants that belong to a narrower
// scope with its instance intact.
const SCOPE_PRIORITY: readonly ResourceScope[] = RESOURCE_SCOPES.filter(
  (scope) => scope !== "workspace",
);

export type MappedGrants = {
  policies: Policy[];
  unmapped: string[];
};

function matchInstance(template: string, path: string): string | null {
  const templateSegments = template.split("/");
  const pathSegments = path.split("/");
  if (templateSegments.length !== pathSegments.length) {
    return null;
  }
  let instance: string | null = null;
  for (const [index, segment] of templateSegments.entries()) {
    const candidate = pathSegments[index];
    if (segment === INSTANCE_TOKEN) {
      instance = candidate;
      continue;
    }
    if (segment !== candidate) {
      return null;
    }
  }
  return instance;
}

function scopeTemplates(rows: readonly PermissionRow[]): string[] {
  const templates = new Set<string>();
  for (const row of rows) {
    templates.add(row.path);
    for (const action of ACTIONS) {
      for (const grant of row.actions[action]) {
        templates.add(grant.path);
      }
    }
  }
  return [...templates].filter((template) => template.includes(INSTANCE_TOKEN));
}

function instanceCandidates(catalogue: ScopeCatalogue, paths: readonly string[]): string[] {
  if (catalogue.instanceNoun === null) {
    return [ALL_INSTANCES];
  }
  const templates = scopeTemplates(catalogueRows(catalogue));
  const found = new Set<string>();
  for (const path of paths) {
    for (const template of templates) {
      const instance = matchInstance(template, path);
      if (instance !== null) {
        found.add(instance);
      }
    }
  }
  const named = [...found].filter((instance) => instance !== "*").sort();
  return found.has("*") ? [ALL_INSTANCES, ...named] : named;
}

function selectionKey(selection: PermissionSelection): string {
  return Object.keys(selection)
    .sort()
    .map((rowId) => `${rowId}:${rowActions(selection, rowId).join("+")}`)
    .join("|");
}

type PolicyDraft = {
  scope: ResourceScope;
  instance: string;
  selection: PermissionSelection;
};

function mergeDrafts(drafts: readonly PolicyDraft[]): Policy[] {
  const merged = new Map<string, Policy>();
  for (const [index, draft] of drafts.entries()) {
    // An all-instances draft already covers every instance, so it stands alone.
    const key =
      draft.instance === ALL_INSTANCES
        ? `all:${index}`
        : `${draft.scope}|${selectionKey(draft.selection)}`;
    const policy = merged.get(key);
    if (policy) {
      policy.instances.push(draft.instance);
      continue;
    }
    merged.set(key, {
      scope: draft.scope,
      instances: [draft.instance],
      selection: draft.selection,
    });
  }
  return [...merged.values()];
}

export function grantsToPolicies(workspaceId: string, grants: readonly string[]): MappedGrants {
  const unique = [...new Set(grants)];
  const mine = (grant: string) => parseUrnPermissionParts(grant)?.workspaceId === workspaceId;
  const remaining = new Set(unique.filter(mine));
  const drafts: PolicyDraft[] = [];

  for (const scope of SCOPE_PRIORITY) {
    const catalogue = CATALOGUES[scope];
    const paths = [...remaining].flatMap((grant) => {
      const parsed = parseUrnPermissionParts(grant);
      return parsed === null ? [] : [parsed.resourcePath];
    });

    for (const instance of instanceCandidates(catalogue, paths)) {
      let selection: PermissionSelection = {};
      const consumed: string[] = [];
      for (const row of catalogueRows(catalogue)) {
        for (const action of ACTIONS) {
          const urns = rowActionUrns(workspaceId, row, action, instance);
          if (urns.length === 0 || !urns.every((urn) => remaining.has(urn))) {
            continue;
          }
          selection = setRowActions(selection, row.id, [...rowActions(selection, row.id), action]);
          consumed.push(...urns);
        }
      }
      if (consumed.length === 0) {
        continue;
      }
      for (const urn of consumed) {
        remaining.delete(urn);
      }
      drafts.push({ scope, instance, selection });
    }
  }

  return {
    policies: mergeDrafts(drafts),
    unmapped: unique.filter((grant) => remaining.has(grant) || !mine(grant)),
  };
}
