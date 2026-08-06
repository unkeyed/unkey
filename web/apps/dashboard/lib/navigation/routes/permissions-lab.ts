/**
 * Route builders for the /permissions-lab prototype area. The concept pages
 * are static routes, so `concept` takes the segment as a literal union and
 * buildRoute validates each resulting pattern against the generated route
 * table.
 */
import type { Route } from "next";
import { type WorkspaceScope, buildRoute } from "./shared";

export const PERMISSIONS_LAB_CONCEPTS = [
  "omnibox",
  "blast-radius",
  "share-sheet",
  "scope-picker",
  "as-code",
  "debugger",
  "changesets",
  "recipes",
  "matrix",
  "everywhere",
] as const;

export type PermissionsLabConcept = (typeof PERMISSIONS_LAB_CONCEPTS)[number];

export const permissionsLabRoutes = {
  index({ workspaceSlug }: WorkspaceScope): Route {
    return buildRoute("/[workspaceSlug]/permissions-lab", { workspaceSlug });
  },

  concept({ workspaceSlug }: WorkspaceScope, segment: PermissionsLabConcept): Route {
    return buildRoute(`/[workspaceSlug]/permissions-lab/${segment}`, { workspaceSlug });
  },
};
