import { describe, expect, test } from "vitest";
import type { StepsData } from "../use-deployment-status";
import { type StagesInput, resolveStages } from "./stages-model";

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

  test("counts assigned domains", () => {
    const steps: StepsData = { building: step(), deploying: step(), network: step() };
    expect(
      byKey(resolveStages(input({ steps, domainCount: 1 }))).domains.resolution.description,
    ).toBe("1 domain assigned");
    expect(
      byKey(resolveStages(input({ steps, domainCount: 3 }))).domains.resolution.description,
    ).toBe("3 domains assigned");
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
