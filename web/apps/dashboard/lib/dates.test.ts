import { describe, expect, it } from "vitest";
import { addDays } from "./dates";

describe("addDays", () => {
  it("rolls over month and year boundaries", () => {
    expect(addDays(new Date(2026, 0, 31, 9, 30), 1)).toEqual(new Date(2026, 1, 1, 9, 30));
    expect(addDays(new Date(2025, 11, 25), 7)).toEqual(new Date(2026, 0, 1));
    expect(addDays(new Date(2026, 2, 1), -1)).toEqual(new Date(2026, 1, 28));
  });

  it("keeps the wall-clock time across a DST change and leaves the input untouched", () => {
    const input = new Date(2026, 2, 7, 9, 30);
    expect(addDays(input, 2).getHours()).toBe(9);
    expect(input).toEqual(new Date(2026, 2, 7, 9, 30));
  });
});
