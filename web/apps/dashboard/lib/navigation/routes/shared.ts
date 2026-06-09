/**
 * Scope types shared across route-builder areas. Each area (projects, apis,
 * ratelimits) extends WorkspaceScope with its own ids.
 */
export type WorkspaceScope = { workspaceSlug: string };
