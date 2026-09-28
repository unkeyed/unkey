import { describe, expect, it } from "vitest";
import { shouldShowLastExit } from ".";

describe("deployment failure badges", () => {
  it("clears an active startup failure after recovery", () => {
    const lastExit = {
      restartCount: 1,
      exitCode: 1,
      signal: null,
      reason: "Error",
      finishedAt: Date.UTC(2026, 8, 8, 12, 0),
      statusReason: null,
      statusMessage: null,
    };

    expect(shouldShowLastExit({ lastExit, status: "deploying" })).toBe(true);
    expect(shouldShowLastExit({ lastExit, status: "ready" })).toBe(false);
    expect(shouldShowLastExit({ lastExit, status: "superseded" })).toBe(false);
    expect(
      shouldShowLastExit({
        lastExit: { ...lastExit, statusReason: "ErrImagePull" },
        status: "ready",
      }),
    ).toBe(true);
    expect(shouldShowLastExit({ lastExit: null, status: "failed" })).toBe(false);
  });
});
