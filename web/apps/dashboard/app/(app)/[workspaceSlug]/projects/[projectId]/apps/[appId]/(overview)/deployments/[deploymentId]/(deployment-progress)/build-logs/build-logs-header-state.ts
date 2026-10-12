import { match } from "@unkey/match";
import type { BuildLogLine } from "./build-log-lines";

export const JUMP_TONES = ["error", "warning"] as const;
export type JumpTone = (typeof JUMP_TONES)[number];
export type JumpCursor = { tone: JumpTone; position: number };
export type PendingJump = JumpTone | "last-error" | "latest";

export type HeaderLabel =
  | { type: "tone"; tone: JumpTone; lines: number }
  | { type: "truncated"; shown: number; total: number }
  | { type: "count"; total: number; hasMore: boolean };

export type ToneCounter =
  | { type: "idle"; tone: JumpTone; count: number; hasMore: boolean }
  | { type: "pending"; tone: JumpTone; count: number; hasMore: boolean }
  | { type: "active"; tone: JumpTone; count: number; hasMore: boolean; position: number };

export type ScrollToLatest = { type: "hidden" } | { type: "ready" } | { type: "pending" };

export type TailRow = { type: "none" } | { type: "loading-more" } | { type: "running" };

export type ToneIndices = Record<JumpTone, number[]>;

export function toToneIndices(lines: BuildLogLine[]): ToneIndices {
  const indices: ToneIndices = { error: [], warning: [] };
  lines.forEach((line, index) => {
    if (line.kind === "entry" && (line.tone === "error" || line.tone === "warning")) {
      indices[line.tone].push(index);
    }
  });
  return indices;
}

// A failed build that is still loading may have its error on a page not read
// yet, so its error counter shows before any error line arrives
export function buildLogsHeaderState({
  toneIndices,
  cursor,
  pendingJump,
  hasMore,
  isFailed,
  entriesTotal,
  entriesShownMax,
}: {
  toneIndices: ToneIndices;
  cursor: JumpCursor | null;
  pendingJump: PendingJump | null;
  hasMore: boolean;
  isFailed: boolean;
  entriesTotal: number;
  entriesShownMax: number;
}): { label: HeaderLabel; counters: ToneCounter[] } {
  const counters = JUMP_TONES.flatMap((tone): ToneCounter[] => {
    const count = toneIndices[tone].length;
    if (count === 0 && !(tone === "error" && isFailed && hasMore)) {
      return [];
    }
    if (cursor?.tone === tone) {
      return [{ type: "active", tone, count, hasMore, position: cursor.position }];
    }
    return [{ type: pendingJump === tone ? "pending" : "idle", tone, count, hasMore }];
  });

  return {
    label: headerLabel(toneIndices, cursor, hasMore, entriesTotal, entriesShownMax),
    counters,
  };
}

function headerLabel(
  toneIndices: ToneIndices,
  cursor: JumpCursor | null,
  hasMore: boolean,
  entriesTotal: number,
  entriesShownMax: number,
): HeaderLabel {
  if (cursor) {
    return { type: "tone", tone: cursor.tone, lines: toneIndices[cursor.tone].length };
  }
  if (entriesTotal > entriesShownMax) {
    return { type: "truncated", shown: entriesShownMax, total: entriesTotal };
  }
  return { type: "count", total: entriesTotal, hasMore };
}

export function headerLabelText(label: HeaderLabel): string {
  return match(label)
    .with(
      { type: "tone" },
      ({ tone, lines }) => `${lines.toLocaleString()} ${tone} ${plural(lines)}`,
    )
    .with(
      { type: "truncated" },
      ({ shown, total }) =>
        `Showing the last ${shown.toLocaleString()} of ${total.toLocaleString()} lines`,
    )
    .with({ type: "count" }, ({ total, hasMore }) =>
      hasMore ? `${total.toLocaleString()}+ lines` : `${total.toLocaleString()} ${plural(total)}`,
    )
    .exhaustive();
}

export function tailRow({
  hasMore,
  isBuilding,
}: { hasMore: boolean; isBuilding: boolean }): TailRow {
  if (hasMore) {
    return { type: "loading-more" };
  }
  return isBuilding ? { type: "running" } : { type: "none" };
}

export function scrollToLatest({
  followsTail,
  hasMore,
  isAtBottom,
  pendingJump,
}: {
  followsTail: boolean;
  hasMore: boolean;
  isAtBottom: boolean;
  pendingJump: PendingJump | null;
}): ScrollToLatest {
  if (followsTail || (!hasMore && isAtBottom)) {
    return { type: "hidden" };
  }
  return pendingJump === "latest" ? { type: "pending" } : { type: "ready" };
}

function plural(count: number): string {
  return count === 1 ? "line" : "lines";
}
