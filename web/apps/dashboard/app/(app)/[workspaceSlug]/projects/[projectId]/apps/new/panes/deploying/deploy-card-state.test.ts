import { describe, expect, it } from "vitest";
import {
  isInstanceCrash,
  resultRows,
  stageRows,
  watchFooter,
  watchStatus,
} from "./deploy-card-state";
import { runView } from "./run-model";

const base = { source: "git" as const, buildError: null, now: 20_000 };

describe("stageRows", () => {
  it("lets only Build Logs expand while a deploy runs", () => {
    const view = runView({
      ...base,
      status: "building",
      health: null,
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
    const crashedView = runView({
      ...base,
      status: "ready",
      health: {
        running: 0,
        unhealthy: true,
        error: "Unhealthy · Instance stopped unexpectedly.",
      },
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
    const unhealthy = {
      state: "unhealthy" as const,
      label: "Unhealthy",
      description: "Instance stopped unexpectedly.",
      tone: "error" as const,
      running: 0,
      total: 1,
      text: "Unhealthy · Instance stopped unexpectedly.",
      lastExit: null,
    };
    expect(isInstanceCrash(crashedView, unhealthy)).toBe(true);
    expect(isInstanceCrash(crashedView, null)).toBe(false);
  });

  it("puts the build error above the logs when the build fails", () => {
    const view = runView({
      ...base,
      status: "failed",
      health: null,
      buildError: "exit code 1",
      steps: { building: { startedAt: 0, endedAt: 10, error: "exit code 1" } },
    });
    expect(stageRows(view, false)[1].detail).toEqual({ type: "logs", error: "exit code 1" });
  });
});

describe("watch copy and footer", () => {
  it("maps every outcome", () => {
    expect(watchFooter).toEqual({
      running: "none",
      live: "continue",
      failed: "failed",
      blocked: "blocked",
    });
    expect(watchStatus.failed).toBe("Deployment failed. Open the failed step to see why.");
  });
});

describe("resultRows", () => {
  it("lists the domain, instances and commit from real data", () => {
    expect(
      resultRows({
        hasDomain: true,
        instances: { text: "1 of 1 running", tone: "plain" },
        gitBranch: "main",
        gitCommitSha: "99766a341c7e",
      }),
    ).toEqual([
      { label: "Domain", value: { type: "domain" } },
      { label: "Instances", value: { type: "mono", text: "1 of 1 running", tone: "plain" } },
      { label: "Commit", value: { type: "mono", text: "main · 99766a3", tone: "plain" } },
    ]);
  });

  it("shows a pending domain and drops rows without data", () => {
    expect(
      resultRows({ hasDomain: false, instances: null, gitBranch: null, gitCommitSha: null }),
    ).toEqual([{ label: "Domain", value: { type: "pending", text: "Assigning domain…" } }]);
  });
});
