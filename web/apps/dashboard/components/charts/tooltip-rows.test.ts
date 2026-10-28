import { describe, expect, it } from "vitest";
import { tooltipRows } from "./tooltip-rows";

describe("tooltipRows", () => {
  it("drops keys without a finite value", () => {
    expect(tooltipRows({ a: 1, b: undefined, c: Number.NaN }, ["a", "b", "c", "d"])).toEqual({
      rows: [{ key: "a", value: 1 }],
      hiddenCount: 0,
    });
  });

  it("sorts rows by value, highest first", () => {
    expect(tooltipRows({ a: 1, b: 3, c: 0 }, ["a", "b", "c"])).toEqual({
      rows: [
        { key: "b", value: 3 },
        { key: "a", value: 1 },
        { key: "c", value: 0 },
      ],
      hiddenCount: 0,
    });
  });

  it("caps rows and counts the rest", () => {
    expect(
      tooltipRows({ a: 1, b: 2, c: 3, d: 4, e: undefined }, ["a", "b", "c", "d", "e"], 2),
    ).toEqual({
      rows: [
        { key: "d", value: 4 },
        { key: "c", value: 3 },
      ],
      hiddenCount: 2,
    });
  });

  it("returns no rows for an empty point", () => {
    expect(tooltipRows({}, ["a", "b"])).toEqual({ rows: [], hiddenCount: 0 });
  });
});
