import type { PermissionsLabConcept } from "@/lib/navigation/routes/permissions-lab";

export interface Concept {
  segment: PermissionsLabConcept;
  title: string;
  pitch: string;
}

export const CONCEPTS: Concept[] = [
  {
    segment: "omnibox",
    title: "Omnibox Grant",
    pitch:
      "One input with segment-aware autocomplete. Invalid grammar is impossible to type, not rejected after submit.",
  },
  {
    segment: "blast-radius",
    title: "Blast Radius Preview",
    pitch:
      "A live panel that answers what a pattern actually covers: counts, samples, and diffs while you type.",
  },
  {
    segment: "share-sheet",
    title: "Share-Sheet on the Resource",
    pitch: "Start from a keyspace or project and manage who can touch it, like sharing a doc.",
  },
  {
    segment: "scope-picker",
    title: "Scope Picker Tree",
    pitch:
      "Pick a node in the resource tree, then a depth. The whole wildcard grammar taught through one radio group.",
  },
  {
    segment: "as-code",
    title: "Permissions as Code",
    pitch: "A two-way UI and text view of the same grants, with copyable API calls for every edit.",
  },
  {
    segment: "debugger",
    title: "Access Debugger",
    pitch:
      "Why can or can't this key do X? Allow and deny verdicts with the exact grant that matched, and near-misses explained.",
  },
  {
    segment: "changesets",
    title: "Changesets",
    pitch:
      "Permission edits staged as human-readable diffs with affected principals, apply and revert like deployments.",
  },
  {
    segment: "recipes",
    title: "Role Recipes",
    pitch:
      "Curated presets with typed holes: pick a keyspace, get real grants. Learn the grammar by example.",
  },
  {
    segment: "matrix",
    title: "The Matrix",
    pitch:
      "Resources by actions as a progressive grid. Wildcard grants render as spanning rows you can materialize.",
  },
  {
    segment: "everywhere",
    title: "No Permissions Page",
    pitch:
      "Grants as inline chips on a key detail page plus a command palette. The permission system dissolves into existing surfaces.",
  },
];
