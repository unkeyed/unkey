import { describe, expect, it } from "vitest";
import { isInstanceCrash, stageRows } from "./deploy-card-state";
import { summarizeInstances } from "./instance-state";
import { runView } from "./run-model";

const base = { source: "git" as const, buildError: null, now: 20_000 };

describe("stageRows", () => {
  it("lets only Build Logs expand while a deploy runs", () => {
    const view = runView({
      ...base,
      status: "building",
      instances: null,
      steps: {
        queued: { startedAt: 0, endedAt: 55, error: null },
        building: { startedAt: 55, endedAt: null, error: null },
      },
    });
    const rows = stageRows(view, false);
    expect(rows.map((row) => [row.title, row.expandable, row.detail.type])).toEqual([
      ["Queued", false, "none"],
      ["Build logs", true, "logs"],
      ["Starting instances", false, "none"],
      ["Assigning domains", false, "none"],
      ["Finalizing", false, "none"],
    ]);
    expect(rows[0].meta).toBe("0s");
    expect(rows[1].detail).toEqual({ type: "logs", error: null });
  });

  it("shows the crash detail for an unhealthy instance and the step error otherwise", () => {
    const unhealthy = summarizeInstances({
      instances: [{ status: "failed" }],
      desiredInstanceCount: 1,
      lastExit: null,
    });
    const crashedView = runView({
      ...base,
      status: "ready",
      instances: unhealthy,
      steps: {
        queued: { startedAt: 0, endedAt: 55, error: null },
        building: { startedAt: 55, endedAt: 1_000, error: null },
        deploying: { startedAt: 1_000, endedAt: 2_000, error: null },
      },
    });
    expect(stageRows(crashedView, true)[2].detail).toEqual({ type: "crash" });
    expect(stageRows(crashedView, false)[2].detail).toEqual({
      type: "error",
      message: "Unhealthy · Instance stopped unexpectedly.",
    });
    expect(stageRows(crashedView, true)[3].meta).toBe("Skipped");
    expect(isInstanceCrash(crashedView, unhealthy)).toBe(true);
    expect(isInstanceCrash(crashedView, null)).toBe(false);
  });

  it("puts the build error above the logs when the build fails", () => {
    const view = runView({
      ...base,
      status: "failed",
      instances: null,
      buildError: "exit code 1",
      steps: { building: { startedAt: 0, endedAt: 10, error: "exit code 1" } },
    });
    expect(stageRows(view, false)[1].detail).toEqual({ type: "logs", error: "exit code 1" });
  });
});
