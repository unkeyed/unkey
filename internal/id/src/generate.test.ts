import { afterEach, beforeEach, describe } from "node:test";
import { expect, test, vi } from "vitest";
import { newId } from "./generate";

beforeEach(() => {
  vi.useFakeTimers();
});
afterEach(() => {
  vi.useRealTimers();
});
describe("ids are k-sorted by time", () => {
  const testCases = [
    {
      k: 2,
      n: 1_000,
    },
    {
      k: 10,
      n: 10_000,
    },
  ];

  for (const tc of testCases) {
    test(`k: ${tc.k}, n: ${tc.n}`, () => {
      const ids = new Array(tc.n).fill(null).map((_, i) => {
        vi.setSystemTime(new Date(i * 10000));

        return newId("test");
      });
      const sorted = [...ids].sort();

      for (let i = 0; i < ids.length; i++) {
        expect(Math.abs(ids.indexOf(sorted[i]) - i)).toBeLessThanOrEqual(tc.k);
      }
    });
  }
});
