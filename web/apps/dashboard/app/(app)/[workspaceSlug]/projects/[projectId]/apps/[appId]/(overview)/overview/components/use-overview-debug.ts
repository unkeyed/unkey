"use client";

import { parseAsBoolean, parseAsStringLiteral, useQueryState } from "nuqs";

// Shared, URL-backed debug state for the overview card. Both the floating nav
// (writer) and the card container (reader) call this hook, so a selected set of
// states is shareable by copying the link. Defaults render the happy Option-G
// path, so a bare /overview URL shows the real card.
export const DEBUG_VIEWS = ["card", "loading", "empty"] as const;
export const DEBUG_STATUSES = ["live", "deploying", "crashing", "failed", "stopped"] as const;
export const DEBUG_DOMAINS = ["custom", "generated", "none"] as const;
export const DEBUG_TRAFFIC = ["normal", "zero"] as const;
export const DEBUG_WINDOWS = ["auto", "hour", "day", "week"] as const;

export type DebugView = (typeof DEBUG_VIEWS)[number];
export type DebugStatus = (typeof DEBUG_STATUSES)[number];
export type DebugDomain = (typeof DEBUG_DOMAINS)[number];
export type DebugTraffic = (typeof DEBUG_TRAFFIC)[number];
export type DebugWindow = (typeof DEBUG_WINDOWS)[number];

const opts = { history: "replace" as const };

export function useOverviewDebug() {
  // Keys are prefixed `dbg` so they never collide with real query params on the
  // same URL — the deployments list below reads `status`/`branch`/`environment`
  // as filter arrays and crashes on a plain-string value.
  const [view, setView] = useQueryState(
    "dbgView",
    parseAsStringLiteral(DEBUG_VIEWS).withDefault("card").withOptions(opts),
  );
  const [status, setStatus] = useQueryState(
    "dbgStatus",
    parseAsStringLiteral(DEBUG_STATUSES).withDefault("live").withOptions(opts),
  );
  const [domain, setDomain] = useQueryState(
    "dbgDomain",
    parseAsStringLiteral(DEBUG_DOMAINS).withDefault("custom").withOptions(opts),
  );
  const [traffic, setTraffic] = useQueryState(
    "dbgTraffic",
    parseAsStringLiteral(DEBUG_TRAFFIC).withDefault("normal").withOptions(opts),
  );
  const [win, setWin] = useQueryState(
    "dbgWin",
    parseAsStringLiteral(DEBUG_WINDOWS).withDefault("auto").withOptions(opts),
  );
  const [rolledBack, setRolledBack] = useQueryState(
    "dbgRolledback",
    parseAsBoolean.withDefault(false).withOptions(opts),
  );

  return {
    view,
    setView,
    status,
    setStatus,
    domain,
    setDomain,
    traffic,
    setTraffic,
    win,
    setWin,
    rolledBack,
    setRolledBack,
  };
}
