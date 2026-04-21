"use client";

import { TopHeader } from "./shared/top-header";

/**
 * v3 — header-led nav (OG Vercel mold). No sidebar at workspace level.
 * Sibling variants accept `workspace` + Sidebar props for dispatching, but
 * v3 doesn't render a sidebar — the header pulls live workspace data
 * itself via `useWorkspaceNavigation`. The layout uses `V3_HEADER_HEIGHT`
 * to push content down past the fixed header.
 */
export function V3Variant() {
  return <TopHeader />;
}
