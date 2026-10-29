import { formatTimestamp } from "@unkey/ui";
import { describe, expect, it } from "vitest";
import { axisEdges } from "./axis-edges";

const NOW = Date.UTC(2026, 9, 26, 15, 0);
const MINUTE = 60_000;
const HOUR = 60 * MINUTE;
const LIVE_END = { kind: "text", text: "now" };

describe("axisEdges", () => {
  it("reads a rolling window of a day or less as an age and now", () => {
    expect(
      axisEdges({ startMs: NOW - 12 * HOUR, endMs: NOW, bucketMs: 5 * MINUTE, nowMs: NOW }),
    ).toEqual([{ kind: "ago", at: NOW - 12 * HOUR }, LIVE_END]);
  });

  it("starts a longer rolling window on its date", () => {
    const startMs = NOW - 7 * 24 * HOUR;
    expect(axisEdges({ startMs, endMs: NOW, bucketMs: HOUR, nowMs: NOW })).toEqual([
      { kind: "text", text: formatTimestamp(startMs, "MMM d") },
      LIVE_END,
    ]);
  });

  it("treats a lagging last bucket as still rolling", () => {
    const [, end] = axisEdges({
      startMs: NOW - HOUR,
      endMs: NOW - 2 * MINUTE,
      bucketMs: MINUTE,
      nowMs: NOW,
    });
    expect(end).toEqual(LIVE_END);
  });

  it("stays rolling when the last settled bucket starts two buckets back", () => {
    const [, end] = axisEdges({
      startMs: NOW - 24 * HOUR,
      endMs: NOW - 30 * MINUTE - 15_000,
      bucketMs: 15 * MINUTE,
      nowMs: NOW,
    });
    expect(end).toEqual(LIVE_END);
  });

  it("reads a past window as wall clock time", () => {
    const startMs = NOW - 6 * 24 * HOUR;
    const endMs = NOW - 5 * 24 * HOUR;
    expect(axisEdges({ startMs, endMs, bucketMs: 15 * MINUTE, nowMs: NOW })).toEqual([
      { kind: "text", text: formatTimestamp(startMs, "MMM d, h:mm a") },
      { kind: "text", text: formatTimestamp(endMs, "MMM d, h:mm a") },
    ]);
  });
});
