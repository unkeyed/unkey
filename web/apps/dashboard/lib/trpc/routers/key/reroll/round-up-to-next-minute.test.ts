import { describe, expect, it } from "vitest";
import { roundUpToNextMinute } from "./round-up-to-next-minute";

describe("roundUpToNextMinute", () => {
  const cases: Array<{ name: string; inputMs: number; expectedMs: number }> = [
    {
      name: "timestamp already on a minute boundary is returned unchanged",
      inputMs: Date.UTC(2026, 3, 21, 12, 30, 0, 0),
      expectedMs: Date.UTC(2026, 3, 21, 12, 30, 0, 0),
    },
    {
      name: "timestamp with seconds rounds up",
      inputMs: Date.UTC(2026, 3, 21, 12, 30, 1, 0),
      expectedMs: Date.UTC(2026, 3, 21, 12, 31, 0, 0),
    },
    {
      name: "timestamp with only milliseconds rounds up",
      inputMs: Date.UTC(2026, 3, 21, 12, 30, 0, 1),
      expectedMs: Date.UTC(2026, 3, 21, 12, 31, 0, 0),
    },
    {
      name: "timestamp near end of minute rounds up",
      inputMs: Date.UTC(2026, 3, 21, 12, 30, 59, 999),
      expectedMs: Date.UTC(2026, 3, 21, 12, 31, 0, 0),
    },
    {
      name: "epoch (0) is returned unchanged",
      inputMs: 0,
      expectedMs: 0,
    },
  ];

  for (const tc of cases) {
    it(tc.name, () => {
      expect(roundUpToNextMinute(new Date(tc.inputMs)).getTime()).toBe(tc.expectedMs);
    });
  }
});
