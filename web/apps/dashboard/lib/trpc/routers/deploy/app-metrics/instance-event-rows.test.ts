import { describe, expect, it } from "vitest";
import { type RawRow, toInstanceEvents } from "./instance-event-rows";

function row(overrides: Partial<RawRow>): RawRow {
  return {
    event_fingerprint: "fp_1",
    time: 1_000,
    pod_name: "checkout-0",
    region: "us-east-1",
    deployment_id: "d_1",
    event_kind: "terminated",
    reason: "Error",
    exit_code: 1,
    message: "",
    ...overrides,
  };
}

const ids = new Map([
  ["checkout-0::us-east-1", "ins_1"],
  ["checkout-0::eu-central-1", "ins_eu"],
]);

describe("toInstanceEvents", () => {
  it("maps kubelet kinds and reasons to event kinds", () => {
    const kinds = toInstanceEvents(
      [
        row({ reason: "OOMKilled", exit_code: 137 }),
        row({ reason: "Error", exit_code: 2 }),
        row({ reason: "Completed", exit_code: 0 }),
        row({ reason: "", exit_code: 0 }),
        row({ event_kind: "waiting", reason: "CrashLoopBackOff", exit_code: 0 }),
        row({ event_kind: "waiting", reason: "ErrImagePull", exit_code: 0 }),
        row({ event_kind: "waiting", reason: "ImagePullBackOff", exit_code: 0 }),
        row({ event_kind: "waiting", reason: "CreateContainerConfigError", exit_code: 0 }),
      ],
      ids,
    ).map((e) => e.kind);
    expect(kinds).toEqual([
      "oom",
      "crashed",
      "completed",
      "completed",
      "crashloop",
      "image-pull",
      "image-pull",
      "config",
    ]);
  });

  it("keeps starts and drops routine waiting reasons", () => {
    expect(
      toInstanceEvents(
        [
          row({ event_kind: "running", reason: "", exit_code: 0 }),
          row({ event_kind: "waiting", reason: "ContainerCreating", exit_code: 0 }),
        ],
        ids,
      ).map((e) => [e.kind, e.exitCode]),
    ).toEqual([["started", null]]);
  });

  it("keeps the exit code only for terminations, trims the message, and resolves the instance id", () => {
    expect(
      toInstanceEvents(
        [
          row({ reason: "OOMKilled", exit_code: 137, message: "  OOM killed\n" }),
          row({
            event_fingerprint: "fp_2",
            pod_name: "checkout-gone",
            event_kind: "waiting",
            reason: "CrashLoopBackOff",
            exit_code: 1,
          }),
        ],
        ids,
      ),
    ).toEqual([
      {
        id: "1000:checkout-0:fp_1",
        atMs: 1_000,
        instance: "checkout-0",
        instanceId: "ins_1",
        region: "us-east-1",
        deploymentId: "d_1",
        kind: "oom",
        reason: "OOMKilled",
        exitCode: 137,
        message: "OOM killed",
      },
      {
        id: "1000:checkout-gone:fp_2",
        atMs: 1_000,
        instance: "checkout-gone",
        instanceId: null,
        region: "us-east-1",
        deploymentId: "d_1",
        kind: "crashloop",
        reason: "CrashLoopBackOff",
        exitCode: null,
        message: null,
      },
    ]);
  });

  it("resolves the instance id by pod name and region", () => {
    expect(
      toInstanceEvents([row({ region: "eu-central-1" }), row({ region: "ap-east-1" })], ids).map(
        (e) => e.instanceId,
      ),
    ).toEqual(["ins_eu", null]);
  });
});
