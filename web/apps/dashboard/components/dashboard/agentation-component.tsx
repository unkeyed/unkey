"use client";

import { Agentation } from "agentation";

/**
 * Visual annotation overlay for AI coding agents.
 * Only mounts in development — the NODE_ENV check at the import site
 * already prevents it loading in production, but we guard again here
 * to make the intent explicit at the render boundary.
 */
export function AgentationOverlay() {
  if (process.env.NODE_ENV !== "development") {
    return null;
  }
  return <Agentation endpoint="http://localhost:4747" />;
}
