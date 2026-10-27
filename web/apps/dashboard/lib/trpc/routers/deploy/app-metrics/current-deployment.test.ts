import { describe, expect, it } from "vitest";
import { type LiveDeployment, currentDeploymentId, deployRole } from "./current-deployment";

const older: LiveDeployment = { id: "d_older", status: "ready", createdAt: 1_000 };
const newer: LiveDeployment = { id: "d_newer", status: "ready", createdAt: 3_000 };
const rolling: LiveDeployment = { id: "d_rolling", status: "deploying", createdAt: 4_000 };

function roles(
  environmentKind: "production" | "preview",
  appCurrentDeploymentId: string | null,
  live: LiveDeployment[],
) {
  const routing =
    environmentKind === "production" ? { currentDeploymentId: appCurrentDeploymentId } : null;
  const currentId = currentDeploymentId({ routing, live });
  return Object.fromEntries(live.map((d) => [d.id, deployRole(d, currentId)]));
}

describe("currentDeploymentId and deployRole", () => {
  it("keeps a newer ready deploy previous when an older one was promoted", () => {
    expect(roles("production", "d_older", [older, newer])).toEqual({
      d_older: "current",
      d_newer: "previous",
    });
  });

  it("marks a deploy still in progress as rolling out", () => {
    expect(roles("production", "d_newer", [older, newer, rolling])).toEqual({
      d_older: "previous",
      d_newer: "current",
      d_rolling: "rolling-out",
    });
  });

  it("has no current deploy in production when the app has none", () => {
    expect(roles("production", null, [older])).toEqual({ d_older: "previous" });
  });

  it("uses the newest ready deploy with live instances in preview", () => {
    expect(roles("preview", "d_older", [older, newer, rolling])).toEqual({
      d_older: "previous",
      d_newer: "current",
      d_rolling: "rolling-out",
    });
  });

  it("never calls a failed newer deploy rolling out", () => {
    const failed: LiveDeployment = { id: "d_failed", status: "failed", createdAt: 5_000 };
    expect(roles("production", "d_older", [older, failed])).toEqual({
      d_older: "current",
      d_failed: "previous",
    });
  });

  it("has no current deploy in preview when none is ready", () => {
    expect(roles("preview", null, [rolling])).toEqual({ d_rolling: "rolling-out" });
  });
});
