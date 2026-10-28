import { formatTimestamp } from "@unkey/ui";

const MINUTE_MS = 60_000;
const DAY_MS = 24 * 60 * MINUTE_MS;

export type AxisEdge = { kind: "ago"; at: number } | { kind: "text"; text: string };

export function axisEdges({
  startMs,
  endMs,
  bucketMs,
  nowMs,
}: {
  startMs: number;
  endMs: number;
  bucketMs: number;
  nowMs: number;
}): [AxisEdge, AxisEdge] {
  const rolling = nowMs - endMs <= Math.max(3 * bucketMs, 5 * MINUTE_MS);
  if (rolling) {
    return [
      endMs - startMs <= DAY_MS
        ? { kind: "ago", at: startMs }
        : { kind: "text", text: formatTimestamp(startMs, "MMM d") },
      { kind: "text", text: "now" },
    ];
  }
  return [
    { kind: "text", text: formatTimestamp(startMs, "MMM d, h:mm a") },
    { kind: "text", text: formatTimestamp(endMs, "MMM d, h:mm a") },
  ];
}
