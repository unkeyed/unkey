import { describe, expect, it } from "vitest";
import { mergeDeploymentTargets, pageTargetDeployments } from "./target-pagination";

describe("connection target deployment pagination", () => {
  it("uses the last included row as the cursor, including tied timestamps", () => {
    const rows = [
      { id: "dep-Z", createdAt: 30 },
      { id: "dep-A", createdAt: 30 },
      { id: "dep-old", createdAt: 10 },
    ];
    expect(pageTargetDeployments(rows, 2)).toEqual({
      page: [rows[0], rows[1]],
      nextCursor: { id: "dep-A", createdAt: 30 },
    });
    expect(pageTargetDeployments(rows, 3)).toEqual({ page: rows, nextCursor: null });
    expect(pageTargetDeployments([], 3)).toEqual({ page: [], nextCursor: null });
  });

  it("keeps a live pin resolved independently when it is beyond the choices page", () => {
    const choices = Array.from({ length: 100 }, (_, index) => ({
      id: `choice-${index}`,
    }));
    const livePin = { id: "older-live-pin", status: "ready" };

    expect(mergeDeploymentTargets([livePin], choices)).toEqual([livePin, ...choices]);
    expect(
      mergeDeploymentTargets(
        [livePin],
        [
          { id: "older-live-pin", status: "stopped" },
          { id: "another", status: "ready" },
        ],
      ),
    ).toEqual([livePin, { id: "another", status: "ready" }]);
  });
});
