import { describe, expect, test } from "vitest";
import type { StepsData } from "../use-deployment-status";
import {
  type StagesInput,
  followFailedStage,
  initialOpenStage,
  resolveStages,
  toggleStage,
} from "./stages-model";

type Step = NonNullable<NonNullable<StepsData>["building"]>;

const step = (overrides: Partial<Step> = {}): Step => ({
  startedAt: 1_000,
  endedAt: 3_000,
  duration: 2_000,
  completed: true,
  error: null,
  ...overrides,
});

const input = (overrides: Partial<StagesInput> = {}): StagesInput => ({
  steps: undefined,
  now: 5_000,
  status: "building",
  isPrebuilt: false,
  instances: null,
  desiredState: "running",
  domainCount: 0,
  ...overrides,
});

const byKey = (stages: ReturnType<typeof resolveStages>) =>
  Object.fromEntries(stages.map((s) => [s.key, s]));

describe("resolveStages", () => {
  test("lists build, instances and domains in order", () => {
    expect(resolveStages(input()).map((s) => s.key)).toEqual(["build", "instances", "domains"]);
  });

  test("waits on every stage before steps load", () => {
    const stages = byKey(resolveStages(input()));
    expect(stages.build.resolution).toEqual({
      status: "pending",
      description: "Waiting to start",
      duration: undefined,
    });
    expect(stages.instances.expandable).toBe(false);
  });

  test("a running build ticks against now", () => {
    const stages = byKey(
      resolveStages(input({ steps: { building: step({ endedAt: null, completed: false }) } })),
    );
    expect(stages.build.resolution).toMatchObject({ status: "started", duration: 4_000 });
  });

  test("a prebuilt image has no build step and reads as prebuilt", () => {
    const stages = byKey(resolveStages(input({ isPrebuilt: true, steps: { deploying: step() } })));
    expect(stages.build.resolution).toMatchObject({
      status: "completed",
      description: "Image was prebuilt",
    });
    expect(stages.build.expandable).toBe(false);
  });

  test("a failed build carries the step error and skips the rest", () => {
    const stages = byKey(
      resolveStages(
        input({
          status: "failed",
          steps: { building: step({ error: "exit 1", completed: false }) },
        }),
      ),
    );
    expect(stages.build.resolution).toMatchObject({ status: "error", description: "exit 1" });
    expect(stages.instances.resolution.status).toBe("skipped");
    expect(stages.domains.resolution.status).toBe("skipped");
  });

  test("unhealthy instances turn a finished deploy into an error", () => {
    const stages = byKey(
      resolveStages(
        input({
          steps: { building: step(), deploying: step() },
          instances: { state: "unhealthy", running: 0, total: 2 },
        }),
      ),
    );
    expect(stages.instances.resolution).toMatchObject({
      status: "error",
      description: "0 of 2 running · Unhealthy",
    });
    expect(stages.instances.expandable).toBe(true);
  });

  test("instances and domains stages drop the in-progress verb once the step stops running", () => {
    const label = (overrides: Partial<StagesInput>) =>
      byKey(resolveStages(input(overrides))).instances.label;
    expect(label({})).toBe("Starting Instances");
    expect(label({ steps: { deploying: step({ endedAt: null, completed: false }) } })).toBe(
      "Starting Instances",
    );
    expect(
      label({
        status: "failed",
        steps: { deploying: step({ error: "image pull failed", completed: false }) },
      }),
    ).toBe("Instances");
    expect(label({ status: "failed" })).toBe("Instances");
    expect(label({ steps: { deploying: step() } })).toBe("Instances");
    expect(
      label({
        steps: { deploying: step() },
        instances: { state: "unhealthy", running: 0, total: 2 },
      }),
    ).toBe("Instances");
    expect(label({ status: "ready" })).toBe("Instances");

    const domainsLabel = (overrides: Partial<StagesInput>) =>
      byKey(resolveStages(input(overrides))).domains.label;
    expect(domainsLabel({})).toBe("Assigning Domains");
    expect(domainsLabel({ steps: { network: step({ endedAt: null, completed: false }) } })).toBe(
      "Assigning Domains",
    );
    expect(domainsLabel({ steps: { network: step() } })).toBe("Domains");
    expect(domainsLabel({ status: "failed" })).toBe("Domains");
  });

  test("counts assigned domains", () => {
    const steps: StepsData = { building: step(), deploying: step(), network: step() };
    expect(
      byKey(resolveStages(input({ steps, domainCount: 1 }))).domains.resolution.description,
    ).toBe("1 domain assigned");
    expect(
      byKey(resolveStages(input({ steps, domainCount: 3 }))).domains.resolution.description,
    ).toBe("3 domains assigned");
  });

  test("domains read as in progress until they load", () => {
    const steps: StepsData = { building: step(), deploying: step(), network: step() };
    expect(byKey(resolveStages(input({ steps, domainCount: null }))).domains.resolution).toEqual({
      status: "completed",
      description: "Assigning domains",
      duration: 2_000,
    });
  });

  test("a stopped deployment without step records reads as finished", () => {
    const stages = byKey(
      resolveStages(input({ status: "stopped", desiredState: "stopped", domainCount: 2 })),
    );
    expect(stages.build.resolution).toEqual({
      status: "completed",
      description: "Build complete",
      duration: undefined,
    });
    expect(stages.instances.resolution).toEqual({
      status: "completed",
      description: "No instances running",
      duration: undefined,
    });
    expect(stages.domains.resolution).toEqual({
      status: "completed",
      description: "2 domains assigned",
      duration: undefined,
    });
  });

  test("a ready deployment without step records reads as complete", () => {
    const stages = byKey(resolveStages(input({ status: "ready", domainCount: 1 })));
    expect(stages.build.resolution.status).toBe("completed");
    expect(stages.instances.resolution.status).toBe("completed");
    expect(stages.domains.resolution).toMatchObject({
      status: "completed",
      description: "1 domain assigned",
    });
  });
});

describe("open stage", () => {
  test("a stage that failed before mount starts open", () => {
    expect(initialOpenStage("build")).toEqual({ open: "build", autoOpened: ["build"] });
    expect(initialOpenStage(null)).toEqual({ open: null, autoOpened: [] });
  });

  test("opens a stage the first time it fails and keeps it open when it recovers", () => {
    const failed = followFailedStage(initialOpenStage(null), "instances");
    expect(failed).toEqual({ open: "instances", autoOpened: ["instances"] });
    expect(followFailedStage(failed, null)).toEqual({
      open: "instances",
      autoOpened: ["instances"],
    });
  });

  test("a flapping stage does not reopen after the user closes it", () => {
    let state = followFailedStage(initialOpenStage(null), "instances");
    state = toggleStage(state, "instances");
    state = followFailedStage(state, null);
    state = followFailedStage(state, "instances");
    expect(state).toEqual({ open: null, autoOpened: ["instances"] });
  });

  test("a flapping stage does not steal focus from the stage the user opened", () => {
    let state = followFailedStage(initialOpenStage(null), "instances");
    state = toggleStage(state, "build");
    state = followFailedStage(state, null);
    state = followFailedStage(state, "instances");
    expect(state.open).toBe("build");
  });

  test("a different stage failing for the first time still opens", () => {
    const state = toggleStage(followFailedStage(initialOpenStage(null), "build"), "build");
    expect(followFailedStage(state, "instances")).toEqual({
      open: "instances",
      autoOpened: ["build", "instances"],
    });
  });
});
