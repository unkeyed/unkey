import type { Deployment } from "@/lib/collections/deploy/deployments";
import { describe, expect, test } from "vitest";
import {
  type InstancesInput,
  instanceDetail,
  resolveInstancesView,
  trackEntered,
} from "./instances-model";

type Instance = Deployment["instances"][number];
type LastExit = NonNullable<Instance["lastExit"]>;

const instance = (id: string, region: string, overrides: Partial<Instance> = {}): Instance => ({
  id,
  region: { id: `reg_${region}`, name: region, platform: "aws" },
  status: "running",
  statusReason: null,
  lastExit: null,
  ...overrides,
});

const exit = (overrides: Partial<LastExit> = {}): LastExit => ({
  restartCount: 0,
  exitCode: null,
  signal: null,
  reason: null,
  finishedAt: null,
  statusReason: null,
  ...overrides,
});

const region = (name: string) => ({ id: `reg_${name}`, name, platform: "aws" });

const input = (overrides: Partial<InstancesInput> = {}): InstancesInput => ({
  instances: [],
  desiredRegions: [],
  targetRegions: [],
  desiredState: "running",
  desiredInstanceCount: 0,
  stage: "completed",
  ...overrides,
});

const diagramOf = (overrides: Partial<InstancesInput>) => {
  const view = resolveInstancesView(input(overrides));
  if (view.kind !== "diagram") {
    throw new Error(`expected a diagram, got ${view.kind}`);
  }
  return view;
};

describe("resolveInstancesView", () => {
  test("groups by region in desired order, unknown regions after by name, instances by id", () => {
    const view = diagramOf({
      instances: [
        instance("ins_c", "us-east-1"),
        instance("ins_b", "ap-south-1"),
        instance("ins_a", "us-east-1"),
        instance("ins_d", "eu-central-1"),
        instance("ins_e", "ap-east-1"),
      ],
      desiredRegions: [{ region: region("eu-central-1") }, { region: region("us-east-1") }],
    });
    expect(
      view.regions.map((r) => [
        r.name,
        r.content.kind === "instances" ? r.content.instances.map((i) => i.id) : r.content.kind,
      ]),
    ).toEqual([
      ["eu-central-1", ["ins_d"]],
      ["us-east-1", ["ins_a", "ins_c"]],
      ["ap-east-1", ["ins_e"]],
      ["ap-south-1", ["ins_b"]],
    ]);
  });

  test("aggregates each region and counts its running instances", () => {
    const view = diagramOf({
      instances: [
        instance("ins_a", "us-east-1"),
        instance("ins_b", "us-east-1", {
          statusReason: "CrashLoopBackOff",
          lastExit: exit({ statusReason: "CrashLoopBackOff", restartCount: 6 }),
        }),
        instance("ins_c", "eu-central-1", { status: "pending" }),
      ],
    });
    expect(view).toEqual({
      kind: "diagram",
      regions: [
        {
          key: "eu-central-1",
          name: "eu-central-1",
          state: "starting",
          running: 0,
          total: 1,
          content: {
            kind: "instances",
            instances: [{ id: "ins_c", state: "starting", detail: null, restarts: 0 }],
            waiting: 0,
          },
        },
        {
          key: "us-east-1",
          name: "us-east-1",
          state: "unhealthy",
          running: 1,
          total: 2,
          content: {
            kind: "instances",
            instances: [
              { id: "ins_a", state: "running", detail: null, restarts: 0 },
              {
                id: "ins_b",
                state: "unhealthy",
                detail: "CrashLoopBackOff · 6 restarts",
                restarts: 6,
              },
            ],
            waiting: 0,
          },
        },
      ],
    });
  });

  test("a target region without instance rows yet shows as waiting", () => {
    const view = diagramOf({
      instances: [instance("ins_a", "us-east-1")],
      targetRegions: [
        { region: region("us-east-1"), replicas: 1 },
        { region: region("ap-southeast-1"), replicas: 2 },
      ],
    });
    expect(view.regions.map((r) => [r.name, r.content.kind, r.total])).toEqual([
      ["ap-southeast-1", "waiting", 2],
      ["us-east-1", "instances", 1],
    ]);
  });

  test("a region short of its target replicas holds waiting slots and is not running", () => {
    const view = diagramOf({
      instances: [instance("ins_a", "us-east-1"), instance("ins_b", "eu-central-1")],
      targetRegions: [
        { region: region("us-east-1"), replicas: 3 },
        { region: region("eu-central-1"), replicas: 1 },
      ],
      desiredRegions: [{ region: region("us-east-1") }, { region: region("eu-central-1") }],
    });
    expect(view).toEqual({
      kind: "diagram",
      regions: [
        {
          key: "us-east-1",
          name: "us-east-1",
          state: "starting",
          running: 1,
          total: 3,
          content: {
            kind: "instances",
            instances: [{ id: "ins_a", state: "running", detail: null, restarts: 0 }],
            waiting: 2,
          },
        },
        {
          key: "eu-central-1",
          name: "eu-central-1",
          state: "running",
          running: 1,
          total: 1,
          content: {
            kind: "instances",
            instances: [{ id: "ins_b", state: "running", detail: null, restarts: 0 }],
            waiting: 0,
          },
        },
      ],
    });
  });

  test("a stopped deployment shows no waiting slots for its target replicas", () => {
    const view = diagramOf({
      desiredState: "stopped",
      instances: [instance("ins_a", "us-east-1", { status: "inactive" })],
      targetRegions: [{ region: region("us-east-1"), replicas: 2 }],
    });
    expect(view.regions.map((r) => [r.state, r.total, r.content])).toEqual([
      [
        "inactive",
        1,
        {
          kind: "instances",
          instances: [{ id: "ins_a", state: "inactive", detail: null, restarts: 0 }],
          waiting: 0,
        },
      ],
    ]);
  });

  test("before the deploy step starts, early rows are ignored for a skeleton of the target", () => {
    const view = diagramOf({
      stage: "pending",
      instances: [instance("ins_a", "us-east-1")],
      targetRegions: [
        { region: region("us-east-1"), replicas: 2 },
        { region: region("eu-central-1"), replicas: 1 },
      ],
    });
    expect(view).toEqual({
      kind: "diagram",
      regions: [
        {
          key: "eu-central-1",
          name: "eu-central-1",
          state: "starting",
          running: 0,
          total: 1,
          content: { kind: "skeleton", count: 1 },
        },
        {
          key: "us-east-1",
          name: "us-east-1",
          state: "starting",
          running: 0,
          total: 2,
          content: { kind: "skeleton", count: 2 },
        },
      ],
    });
  });

  test("without a target, the skeleton is one unnamed region of the desired count", () => {
    expect(diagramOf({ stage: "pending", desiredInstanceCount: 3 }).regions).toEqual([
      {
        key: "expected",
        name: null,
        state: "starting",
        running: 0,
        total: 3,
        content: { kind: "skeleton", count: 3 },
      },
    ]);
    expect(diagramOf({ stage: "started", desiredInstanceCount: 0 })).toEqual({
      kind: "diagram",
      regions: [
        {
          key: "expected",
          name: null,
          state: "starting",
          running: 0,
          total: 1,
          content: { kind: "skeleton", count: 1 },
        },
      ],
    });
  });

  test("no rows after the deploy step reads as empty, worded by the desired state", () => {
    expect(resolveInstancesView(input({ desiredState: "stopped" }))).toEqual({
      kind: "empty",
      message: "No instances running",
    });
    expect(resolveInstancesView(input({ stage: "error" }))).toEqual({
      kind: "empty",
      message: "No instances",
    });
    expect(resolveInstancesView(input({ stage: "started", desiredState: "stopped" }))).toEqual({
      kind: "empty",
      message: "No instances running",
    });
  });
});

describe("instanceDetail", () => {
  const failed = (lastExit: LastExit | null) =>
    instanceDetail(instance("ins_a", "us-east-1", { status: "failed", lastExit }));

  test("has no detail for non-crashed states", () => {
    expect(instanceDetail(instance("ins_a", "us-east-1", { status: "inactive" }))).toBe(null);
    expect(instanceDetail(instance("ins_a", "us-east-1"))).toBe(null);
  });

  test("has no detail with no restarts and no reason", () => {
    expect(failed(null)).toBe(null);
    expect(failed(exit())).toBe(null);
  });

  test("omits a zero restart count but keeps the exit", () => {
    expect(failed(exit({ reason: "OOMKilled", exitCode: 137 }))).toBe("OOMKilled · exit=137");
  });

  test("counts restarts when no reason is known", () => {
    expect(failed(exit({ restartCount: 4 }))).toBe("4 restarts");
    expect(failed(exit({ restartCount: 1, reason: "Error", exitCode: 1 }))).toBe(
      "Error · exit=1 · 1 restart",
    );
  });
});

describe("trackEntered", () => {
  const start = { state: "unhealthy" as const, restarts: 3, entered: null };

  test("marks a state change as entered", () => {
    expect(trackEntered(start, { state: "running", restarts: 3 })).toEqual({
      state: "running",
      restarts: 3,
      entered: "running",
    });
    expect(
      trackEntered(
        { state: "running", restarts: 3, entered: null },
        {
          state: "unhealthy",
          restarts: 4,
        },
      ),
    ).toEqual({ state: "unhealthy", restarts: 4, entered: "unhealthy" });
  });

  test("a return to running that came with a restart is not celebrated", () => {
    expect(trackEntered(start, { state: "running", restarts: 4 })).toEqual({
      state: "running",
      restarts: 4,
      entered: null,
    });
  });

  test("keeps the entered mark while the state holds", () => {
    const entered = { state: "running" as const, restarts: 3, entered: "running" as const };
    expect(trackEntered(entered, { state: "running", restarts: 3 })).toBe(entered);
    expect(trackEntered(entered, { state: "running", restarts: 5 })).toEqual({
      ...entered,
      restarts: 5,
    });
  });
});
