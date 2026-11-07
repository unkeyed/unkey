import { describe, expect, it } from "vitest";
import { LIVE_LOOKBACK_MS, LIVE_ROW_CAP, livePollWindow, mergeLiveRows } from "./live-tail";

const MIN = 60_000;
const now = 1_800_000_000_000;

describe("livePollWindow", () => {
  it("reaches back past the end of history on the first poll", () => {
    expect(
      livePollWindow({
        historyStart: now - 720 * MIN,
        historyEnd: now - 10 * MIN,
        lastPollAt: undefined,
        now,
      }),
    ).toEqual({ startTime: now - 15 * MIN, endTime: now });
  });

  it("reaches back past the previous poll after that", () => {
    expect(
      livePollWindow({
        historyStart: now - 720 * MIN,
        historyEnd: now - 10 * MIN,
        lastPollAt: now - 2000,
        now,
      }),
    ).toEqual({ startTime: now - 2000 - LIVE_LOOKBACK_MS, endTime: now });
  });

  it("never starts before the history window", () => {
    expect(
      livePollWindow({ historyStart: now - 1 * MIN, historyEnd: now, lastPollAt: undefined, now }),
    ).toEqual({ startTime: now - 1 * MIN, endTime: now });
  });
});

type Row = { id: string; time: number };
const keyOf = (row: Row) => row.id;
const timeOf = (row: Row) => row.time;

describe("mergeLiveRows", () => {
  it("keeps rows that a later poll no longer returns", () => {
    expect(mergeLiveRows([{ id: "a", time: 1 }], [], { keyOf, timeOf })).toEqual([
      { id: "a", time: 1 },
    ]);
  });

  it("adds a late row below a newer one and keeps each row once", () => {
    expect(
      mergeLiveRows(
        [{ id: "c", time: 30 }],
        [
          { id: "c", time: 30 },
          { id: "b", time: 20 },
        ],
        { keyOf, timeOf },
      ),
    ).toEqual([
      { id: "c", time: 30 },
      { id: "b", time: 20 },
    ]);
  });

  it("orders rows that share a millisecond by key", () => {
    expect(
      mergeLiveRows(
        [],
        [
          { id: "a", time: 5 },
          { id: "b", time: 5 },
        ],
        { keyOf, timeOf },
      ).map(keyOf),
    ).toEqual(["b", "a"]);
  });

  it("keeps only the newest rows once over the cap", () => {
    const fetched = Array.from({ length: LIVE_ROW_CAP + 10 }, (_, i) => ({ id: `r${i}`, time: i }));
    const merged = mergeLiveRows([], fetched, { keyOf, timeOf });
    expect(merged).toHaveLength(LIVE_ROW_CAP);
    expect(merged[0]).toEqual({ id: `r${LIVE_ROW_CAP + 9}`, time: LIVE_ROW_CAP + 9 });
    expect(merged.at(-1)).toEqual({ id: "r10", time: 10 });
  });
});
