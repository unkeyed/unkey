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
  resolveInstance,
} from "./catalogue.types";
import { type Policy, rowActions, setRowActions } from "./policy";
import { rowActionUrns } from "./urn";

// Scopes claim grants broadest first. Workspace leads because it is the only
// scope that can express a fully wildcarded grant — the GitHub app among them —
// so a key that reaches everything comes back as one card. It cannot steal from
// a narrower scope: its paths wildcard every id, and a policy on one keyspace
// or one project names that id, so the two never match the same grant.
const SCOPE_PRIORITY: readonly ResourceScope[] = RESOURCE_SCOPES;

export type MappedGrants = {
  policies: Policy[];
  unmapped: string[];
};

// The token stands for a whole path down to an instance ("projects/p_1/apps/a_1"),
// so a template is a fixed prefix and suffix with the instance between them.
function matchInstance(template: string, path: string): string | null {
  const [prefix, suffix, ...extra] = template.split(INSTANCE_TOKEN);
  if (suffix === undefined || extra.length > 0) {
    return null;
  }
  if (!path.startsWith(prefix) || !path.endsWith(suffix)) {
    return null;
  }
  const instance = path.slice(prefix.length, path.length - suffix.length);
  return instance.length === 0 ? null : instance;
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
  const named = [...found].filter((instance) => instance !== catalogue.allInstance).sort();
  return found.has(catalogue.allInstance) ? [ALL_INSTANCES, ...named] : named;
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
          const urns = rowActionUrns(
            workspaceId,
            row,
            action,
            resolveInstance(catalogue, instance),
          );
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
