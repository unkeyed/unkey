import { describe, expect, it } from "vitest";
import { mergeDeploymentTargets, pageTargetDeployments } from "./target-pagination";

describe("connection target deployment pagination", () => {
  it("pages within the selected app instead of across newer deployments from other apps", () => {
    const newerOtherAppDeployments = Array.from(
      { length: 101 },
      (_, index) => ({
        id: `other-${index.toString().padStart(3, "0")}`,
        appId: "other-app",
        createdAt: 1_000 + index,
      }),
    );
    const selectedTarget = {
      id: "selected-live",
      appId: "selected-app",
      createdAt: 1,
    };

    const result = pageTargetDeployments(
      [...newerOtherAppDeployments, selectedTarget],
      "selected-app",
      100,
    );

    expect(result.page).toEqual([selectedTarget]);
    expect(result.nextCursor).toBeNull();
  });

  it("keeps a live pin resolved independently when it is beyond the choices page", () => {
    const choices = Array.from({ length: 100 }, (_, index) => ({
      id: `choice-${index}`,
    }));
    const livePin = { id: "older-live-pin" };

    expect(mergeDeploymentTargets([livePin], choices)).toContain(livePin);
  });
});
