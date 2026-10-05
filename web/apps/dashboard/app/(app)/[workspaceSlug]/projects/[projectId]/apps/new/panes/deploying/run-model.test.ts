import { describe, expect, it } from "vitest";
import {
  emptyLogCopy,
  formatOffset,
  formatStageDuration,
  logGroups,
  nextRevealedSteps,
  runView,
} from "./run-model";

const states = (view: ReturnType<typeof runView>) => view.stages.map((s) => s.state);

describe("runView", () => {
  it("marks the running stage active with a live duration", () => {
    const view = runView({
      health: null,
      status: "building",
      source: "git",
      buildError: null,
      now: 5_000,
      steps: {
        queued: { startedAt: 1_000, endedAt: 1_200, error: null },
        building: { startedAt: 1_200, endedAt: null, error: null },
      },
    });
    expect(states(view)).toEqual(["done", "active", "waiting", "waiting", "waiting"]);
    expect(view.stages[1].durationMs).toBe(3_800);
    expect(view.elapsedMs).toBe(4_000);
    expect(view.outcome).toBe("running");
  });

  it("skips the build for a prebuilt image and fails the stage that errored", () => {
    const view = runView({
      health: null,
      status: "failed",
      source: "oci",
      buildError: null,
      now: 99_000,
      steps: {
        queued: { startedAt: 1_000, endedAt: 1_020, error: null },
        deploying: {
          startedAt: 2_000,
          endedAt: 902_000,
          error: "Instances did not become healthy",
        },
      },
    });
    expect(states(view)).toEqual(["done", "skipped", "failed", "skipped", "skipped"]);
    expect(view.stages[1].note).toBe("Prebuilt image");
    expect(view.failedStage?.key).toBe("deploying");
    expect(view.firstError).toBe("Instances did not become healthy");
    expect(view.elapsedMs).toBe(901_000);
  });

  it("prefers the failing build step error", () => {
    const view = runView({
      health: null,
      status: "failed",
      source: "git",
      buildError: "npm ERR! missing script: build",
      now: 9_000,
      steps: { building: { startedAt: 1_000, endedAt: 4_000, error: "Build failed" } },
    });
    expect(view.firstError).toBe("npm ERR! missing script: build");
  });

  it("completes every stage once live even without step rows", () => {
    const view = runView({
      health: null,
      status: "ready",
      source: "git",
      buildError: null,
      now: 0,
      steps: {},
    });
    expect(states(view)).toEqual(["done", "done", "done", "done", "done"]);
    expect(view.elapsedMs).toBeNull();
  });
});

describe("logGroups", () => {
  it("orders build steps and appends the step error and instance output", () => {
    const groups = logGroups(
      [
        {
          step_id: "b",
          name: "RUN pnpm build",
          started_at: 3_000,
          cached: false,
          error: "exit code 1",
          logs: [{ time: 3_500, message: "error TS2304" }],
        },
        { step_id: "a", name: "FROM node:22", started_at: 1_000, cached: true, error: null },
      ],
      [{ time: 6_000, severity: "warn", message: "listening" }],
    );
    expect(groups.map((g) => g.title)).toEqual([
      "FROM node:22 (cached)",
      "RUN pnpm build",
      "Runtime logs",
    ]);
    expect(groups[1].lines).toEqual([
      { id: "1-b-0", offsetMs: 2_500, text: "error TS2304", tone: "plain" },
      { id: "1-b-error", offsetMs: 2_500, text: "exit code 1", tone: "error" },
    ]);
    expect(groups[2].lines[0].tone).toBe("warn");
  });
});

describe("formatOffset", () => {
  it("formats minutes and seconds", () => {
    expect(formatOffset(65_400)).toBe("01:05");
  });
});

describe("formatStageDuration", () => {
  it("shows whole seconds and minutes, never milliseconds", () => {
    expect(formatStageDuration(56)).toBe("0s");
    expect(formatStageDuration(7_400)).toBe("7s");
    expect(formatStageDuration(80_000)).toBe("1m 20s");
  });
});

describe("runView stage order", () => {
  it("marks earlier stages done when a later stage finished first", () => {
    const view = runView({
      health: null,
      status: "network",
      source: "git",
      buildError: null,
      now: 90_000,
      steps: {
        queued: { startedAt: 1_000, endedAt: 1_100, error: null },
        building: { startedAt: 1_100, endedAt: 81_000, error: null },
        deploying: { startedAt: 81_000, endedAt: null, error: null },
        network: { startedAt: 82_000, endedAt: 83_000, error: null },
      },
    });
    expect(states(view)).toEqual(["done", "done", "done", "done", "waiting"]);
  });

  it("marks stages after a failed step as not reached", () => {
    const view = runView({
      health: null,
      status: "failed",
      source: "git",
      buildError: null,
      now: 90_000,
      steps: {
        queued: { startedAt: 1_000, endedAt: 1_100, error: null },
        building: { startedAt: 1_100, endedAt: 81_000, error: "exit code 1" },
        deploying: { startedAt: 81_000, endedAt: 82_000, error: null },
      },
    });
    expect(states(view)).toEqual(["done", "failed", "skipped", "skipped", "skipped"]);
    expect(view.stages[2].note).toBe("Skipped");
  });
});

describe("nextRevealedSteps", () => {
  it("reveals steps that finished in one poll one at a time, in stage order", () => {
    const revealed = {
      queued: { startedAt: 0, endedAt: 55, error: null },
      building: { startedAt: 55, endedAt: 12_855, error: null },
      deploying: { startedAt: 12_855, endedAt: null, error: null },
    };
    const target = {
      ...revealed,
      deploying: { startedAt: 12_855, endedAt: 15_255, error: null },
      network: { startedAt: 15_255, endedAt: 15_323, error: null },
      finalizing: { startedAt: 15_323, endedAt: 15_352, error: null },
    };
    const first = nextRevealedSteps(revealed, target);
    expect(first?.deploying?.endedAt).toBe(15_255);
    expect(first?.network).toBeUndefined();
    const second = first && nextRevealedSteps(first, target);
    expect(second?.network?.endedAt).toBe(15_323);
    expect(second?.finalizing).toBeUndefined();
    const third = second && nextRevealedSteps(second, target);
    expect(third?.finalizing?.endedAt).toBe(15_352);
    expect(third && nextRevealedSteps(third, target)).toBeNull();
  });
});

describe("runView with instance health", () => {
  const allEnded = {
    queued: { startedAt: 0, endedAt: 55, error: null },
    building: { startedAt: 55, endedAt: 12_855, error: null },
    deploying: { startedAt: 12_855, endedAt: 15_255, error: null },
    network: { startedAt: 15_255, endedAt: 15_323, error: null },
    finalizing: { startedAt: 15_323, endedAt: 15_352, error: null },
  };
  const unhealthy = {
    running: 0,
    unhealthy: true,
    error: "Unhealthy · Instance stopped unexpectedly.",
  };

  it("is live only when the deployment is ready and an instance is running", () => {
    const view = runView({
      status: "ready",
      health: { running: 1, unhealthy: false, error: "" },
      source: "git",
      buildError: null,
      now: 20_000,
      steps: allEnded,
    });
    expect(view.outcome).toBe("live");
    expect(states(view)).toEqual(["done", "done", "done", "done", "done"]);
  });

  it("fails at Starting Instances when a ready deployment's instance is unhealthy", () => {
    const view = runView({
      status: "ready",
      health: unhealthy,
      source: "git",
      buildError: null,
      now: 20_000,
      steps: allEnded,
    });
    expect(view.outcome).toBe("failed");
    expect(states(view)).toEqual(["done", "done", "failed", "skipped", "skipped"]);
    expect(view.stages[3].note).toBe("Skipped");
    expect(view.firstError).toBe("Unhealthy · Instance stopped unexpectedly.");
  });

  it("keeps Starting Instances active until an instance is running", () => {
    const view = runView({
      status: "ready",
      health: { running: 0, unhealthy: false, error: "" },
      source: "git",
      buildError: null,
      now: 20_000,
      steps: {
        queued: allEnded.queued,
        building: allEnded.building,
        deploying: allEnded.deploying,
      },
    });
    expect(view.outcome).toBe("running");
    expect(states(view)).toEqual(["done", "done", "active", "waiting", "waiting"]);
    expect(view.stages[2].durationMs).toBe(7_145);
  });
});

describe("runView before any step starts", () => {
  it("fails at Queued with the status reason when no step ever ran", () => {
    const view = runView({
      status: "failed",
      health: null,
      source: "git",
      buildError: null,
      now: 0,
      steps: {},
    });
    expect(states(view)).toEqual(["failed", "skipped", "skipped", "skipped", "skipped"]);
    expect(view.firstError).toBe("The deployment failed before it started building.");
  });

  it("names a superseded deployment", () => {
    const view = runView({
      status: "superseded",
      health: null,
      source: "git",
      buildError: null,
      now: 0,
      steps: {},
    });
    expect(view.failedStage?.error).toBe(
      "A newer deployment replaced this one before it started building.",
    );
  });
});

describe("emptyLogCopy", () => {
  it("explains an empty log per source and outcome", () => {
    expect(emptyLogCopy("git", "running")).toEqual({
      title: "Waiting for logs…",
      reason: "Logs stream in as soon as the build starts.",
    });
    expect(emptyLogCopy("git", "failed").reason).toBe(
      "The build stopped before it wrote any logs.",
    );
    expect(emptyLogCopy("oci", "live")).toEqual({
      title: "No logs",
      reason: "Images skip the build step. Runtime logs show here once an instance starts.",
    });
  });
});
