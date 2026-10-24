import { describe, expect, it } from "vitest";
import {
  RANGE_PLANS,
  RANGE_PRESETS,
  clampedGrid,
  findRollup,
  latestGrid,
  nextTickMs,
  pickRollup,
  planGrid,
} from "./grid";

const at = (iso: string) => Date.parse(iso);

describe("planGrid", () => {
  it.each([
    ["1h", "1m", 60_000, "2026-10-19T11:52:00Z", "2026-10-19T12:52:00Z", 60],
    ["6h", "5m", 300_000, "2026-10-19T06:50:00Z", "2026-10-19T12:50:00Z", 72],
    ["24h", "15m", 900_000, "2026-10-18T12:45:00Z", "2026-10-19T12:45:00Z", 96],
    ["3d", "1h", 3_600_000, "2026-10-16T12:00:00Z", "2026-10-19T12:00:00Z", 72],
    ["7d", "1h", 3_600_000, "2026-10-12T12:00:00Z", "2026-10-19T12:00:00Z", 168],
    ["14d", "2h", 7_200_000, "2026-10-05T12:00:00Z", "2026-10-19T12:00:00Z", 168],
    ["30d", "6h", 21_600_000, "2026-09-19T12:00:00Z", "2026-10-19T12:00:00Z", 120],
  ] as const)("plans %s as %s buckets", (range, bucket, bucketMs, start, end, buckets) => {
    const grid = planGrid(range, at("2026-10-19T12:52:10Z"));
    expect(grid).toEqual({
      range,
      bucket,
      bucketMs,
      startMs: at(start),
      endMs: at(end),
    });
    expect((grid.endMs - grid.startMs) / bucketMs).toBe(buckets);
  });

  it("keeps an endMs that is already on a bucket edge", () => {
    expect(planGrid("24h", at("2026-10-19T12:45:00Z")).endMs).toBe(at("2026-10-19T12:45:00Z"));
    expect(planGrid("24h", at("2026-10-19T12:44:59.999Z")).endMs).toBe(at("2026-10-19T12:30:00Z"));
  });

  it("is idempotent on its own endMs", () => {
    const grid = planGrid("14d", at("2026-10-19T13:59:59Z"));
    expect(planGrid("14d", grid.endMs)).toEqual(grid);
  });

  it("aligns 2h and 6h buckets to UTC epoch multiples", () => {
    expect(planGrid("14d", at("2026-10-19T01:30:00Z")).endMs).toBe(at("2026-10-19T00:00:00Z"));
    expect(planGrid("30d", at("2026-10-19T17:59:00Z")).endMs).toBe(at("2026-10-19T12:00:00Z"));
  });
});

describe("RANGE_PLANS", () => {
  it("polls only the ranges up to a day", () => {
    expect(RANGE_PRESETS.filter((range) => RANGE_PLANS[range].live)).toEqual(["1h", "6h", "24h"]);
  });
});

describe("latestGrid", () => {
  it("excludes a bucket until 15s after it closes", () => {
    expect(latestGrid("1h", at("2026-10-19T12:45:14.999Z")).endMs).toBe(at("2026-10-19T12:44:00Z"));
    expect(latestGrid("1h", at("2026-10-19T12:45:15Z")).endMs).toBe(at("2026-10-19T12:45:00Z"));
  });

  it("wakes the clock when the next bucket settles", () => {
    const grid = latestGrid("6h", at("2026-10-19T12:47:00Z"));
    expect(nextTickMs(grid)).toBe(at("2026-10-19T12:50:15Z"));
    expect(latestGrid("6h", nextTickMs(grid)).endMs).toBe(at("2026-10-19T12:50:00Z"));
  });
});

describe("clampedGrid", () => {
  const now = at("2026-10-19T12:45:20Z");

  it("clamps a future endMs to the settled edge", () => {
    expect(clampedGrid("1h", at("2026-10-19T14:00:00Z"), now).endMs).toBe(
      at("2026-10-19T12:45:00Z"),
    );
  });

  it("keeps a past endMs", () => {
    expect(clampedGrid("1h", at("2026-10-19T10:00:00Z"), now).endMs).toBe(
      at("2026-10-19T10:00:00Z"),
    );
  });
});

describe("pickRollup", () => {
  const day = 24 * 60 * 60 * 1000;
  const rollups = [
    { table: "per_minute", bucketMs: 60_000, ttlMs: 14 * day },
    { table: "per_15m", bucketMs: 900_000, ttlMs: 30 * day },
    { table: "per_hour", bucketMs: 3_600_000, ttlMs: 90 * day },
  ];
  const now = at("2026-10-19T12:45:20Z");

  it.each([
    ["1h", "per_minute"],
    ["6h", "per_minute"],
    ["24h", "per_15m"],
    ["14d", "per_hour"],
    ["30d", "per_hour"],
  ] as const)("reads %s from %s", (range, table) => {
    expect(pickRollup(rollups, latestGrid(range, now), now).table).toBe(table);
  });

  it("finds nothing when no rollup keeps the window", () => {
    expect(findRollup(rollups.slice(0, 1), latestGrid("30d", now), now)).toBeNull();
  });

  it("throws when no rollup keeps the window", () => {
    expect(() => pickRollup(rollups.slice(0, 1), latestGrid("30d", now), now)).toThrow(
      "No rollup serves a 6h grid over 30d",
    );
  });
});
