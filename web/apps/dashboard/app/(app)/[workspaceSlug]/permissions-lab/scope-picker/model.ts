/**
 * Tree and scope derivation for the scope picker. The tree is derived from
 * ALL_RESOURCES paths: id segments become selectable resource nodes, collection
 * segments (keyspaces, keys, apps, ...) become group headers. Scope options map
 * a selected node to the three patterns the grammar allows without ever showing
 * the user raw wildcard syntax first.
 */

import { RESOURCE_TYPES } from "../lib/catalog";
import { ALL_RESOURCES, type ConcreteResource, labelForPath } from "../lib/mock-data";

export interface TreeNode {
  segment: string;
  path: string;
  /** set when the path names a concrete resource; null for collection groups */
  resource: ConcreteResource | null;
  children: TreeNode[];
}

export function buildResourceTree(): TreeNode[] {
  const roots: TreeNode[] = [];
  const byPath = new Map<string, TreeNode>();

  for (const resource of ALL_RESOURCES) {
    const segments = resource.path.split("/");
    let siblings = roots;
    let path = "";
    for (const segment of segments) {
      path = path === "" ? segment : `${path}/${segment}`;
      let node = byPath.get(path);
      if (!node) {
        node = { segment, path, resource: null, children: [] };
        byPath.set(path, node);
        siblings.push(node);
      }
      siblings = node.children;
    }
    const leaf = byPath.get(resource.path);
    if (leaf) {
      leaf.resource = resource;
    }
  }

  return roots;
}

export function findNode(nodes: TreeNode[], path: string): TreeNode | null {
  for (const node of nodes) {
    if (node.path === path) {
      return node;
    }
    const hit = findNode(node.children, path);
    if (hit) {
      return hit;
    }
  }
  return null;
}

const COLLECTION_NAMES: Record<string, string> = {
  rbac: "RBAC",
};

export function collectionName(segment: string): string {
  return COLLECTION_NAMES[segment] ?? segment.charAt(0).toUpperCase() + segment.slice(1);
}

/** Direct resource children of a collection node. */
export function collectionCount(node: TreeNode): number {
  return node.children.filter((c) => c.resource !== null).length;
}

/** "18 keys", "1 domain": the collection segment doubles as the noun. */
export function countLabel(segment: string, count: number): string {
  const noun = count === 1 && segment.endsWith("s") ? segment.slice(0, -1) : segment;
  return `${count} ${noun}`;
}

/** Collapsed summary on a resource row, e.g. "18 keys" or "3 deployments · 1 domain". */
export function childSummary(node: TreeNode): string | null {
  const parts = node.children
    .filter((c) => c.resource === null)
    .map((c) => countLabel(c.segment, collectionCount(c)));
  return parts.length > 0 ? parts.join(" · ") : null;
}

export type ScopeID = "exact" | "any" | "descendants";

export interface ScopeOption {
  id: ScopeID;
  title: string;
  detail: string;
  pattern: string;
  /** pattern segment index that differs from the concrete path; null for the exact scope */
  changedSegment: number | null;
  disabledReason: string | null;
}

/** Types that can contain nested resources even when they are empty today. */
const CONTAINER_TYPES = new Set(["keyspace", "project", "app", "environment"]);

export function scopeOptionsFor(node: TreeNode): ScopeOption[] {
  const resource = node.resource;
  if (!resource) {
    return [];
  }

  const typeDef = RESOURCE_TYPES.find((t) => t.type === resource.type);
  const typeNoun = (typeDef?.label ?? resource.type).toLowerCase();
  const segments = resource.path.split("/");

  // Nearest ancestor that is itself a resource; top-level resources fall back
  // to the workspace as their parent.
  let parentLabel: string | null = null;
  for (let k = segments.length - 1; k > 0; k--) {
    const prefix = segments.slice(0, k).join("/");
    if (ALL_RESOURCES.some((r) => r.path === prefix)) {
      parentLabel = labelForPath(prefix);
      break;
    }
  }

  const anyPattern = [...segments.slice(0, -1), "*"].join("/");
  const canDescend = node.children.length > 0 || CONTAINER_TYPES.has(resource.type);

  return [
    {
      id: "exact",
      title: `Just this ${typeNoun}`,
      detail: `Applies to ${resource.label} and nothing else.`,
      pattern: resource.path,
      changedSegment: null,
      disabledReason: null,
    },
    {
      id: "any",
      title: parentLabel
        ? `Any ${typeNoun} in ${parentLabel}`
        : `Any ${typeNoun} in this workspace`,
      detail: "The id becomes a wildcard, so this also covers ones created later.",
      pattern: anyPattern,
      changedSegment: segments.length - 1,
      disabledReason: null,
    },
    {
      id: "descendants",
      title: `This ${typeNoun} and everything inside it`,
      detail: `Covers ${resource.label} plus every resource nested under it, now and in the future.`,
      pattern: `${resource.path}/**`,
      changedSegment: segments.length,
      disabledReason: canDescend
        ? null
        : `A ${typeNoun} has no nested resources, so "just this ${typeNoun}" already covers everything.`,
    },
  ];
}
