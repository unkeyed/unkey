import { parseUrnPermissionParts } from "@unkey/rbac";
import { describePermission, humaniseAction } from "./describe-permission";

export type GrantLabel = {
  path: string | null;
  action: string;
};

export function grantLabel(grant: string): GrantLabel {
  const parsed = parseUrnPermissionParts(grant);
  if (parsed === null) {
    return { path: null, action: describePermission(grant) };
  }
  return { path: parsed.resourcePath, action: humaniseAction(parsed.action) };
}

const WILDCARD_SEGMENTS = new Set(["*", "**"]);
const CELL_SEPARATOR = " · ";

// Resource paths always end in an instance slot ("identities/*",
// "keyspaces/ks_1/keys/*"), so the collection a grant acts on is the segment
// before it.
function collectionSegment(resourcePath: string): string | null {
  const collection = resourcePath.split("/").at(-2);
  if (collection === undefined || WILDCARD_SEGMENTS.has(collection)) {
    return null;
  }
  return collection;
}

function singular(word: string): string {
  return word.endsWith("s") ? word.slice(0, -1) : word;
}

// An action usually repeats the collection it acts on ("read_namespace" on
// ".../namespaces/*"). Drop the repeat so a cell reads "Namespaces · Read".
// Only for a bare verb-plus-resource pair: longer actions such as
// "add_permission_to_key" lose their meaning without the last word.
function terseAction(action: string, collection: string): string {
  const words = action.split("_");
  if (words.length === 2 && singular(words[1]) === singular(collection)) {
    return humaniseAction(words[0]);
  }
  return humaniseAction(action);
}

// One-line form for table cells: a Title Case collection noun plus the action,
// never the URN prefix or the workspace id.
export function grantCellLabel(grant: string): string {
  const parsed = parseUrnPermissionParts(grant);
  if (parsed === null) {
    return describePermission(grant);
  }
  const collection = collectionSegment(parsed.resourcePath);
  if (collection === null) {
    return parsed.action === "*" ? "All permissions" : humaniseAction(parsed.action);
  }
  return `${humaniseAction(collection)}${CELL_SEPARATOR}${terseAction(parsed.action, collection)}`;
}

export function grantCellLabels(grants: readonly string[]): string[] {
  return [...new Set(grants.map(grantCellLabel))];
}
