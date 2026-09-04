import { describe, expect, test } from "vitest";
import {
  DEPLOYMENT_STATUSES,
  DEPLOYMENT_STATUS_GROUPS,
  DEPLOYMENT_STATUS_GROUP_NAMES,
  expandDeploymentStatusGroups,
} from "./deployment-status";

describe("DEPLOYMENT_STATUS_GROUPS", () => {
  test("partitions every raw status into exactly one group", () => {
    const all = Object.values(DEPLOYMENT_STATUS_GROUPS).flat();
    expect([...all].sort()).toEqual([...DEPLOYMENT_STATUSES].sort());
    expect(new Set(all).size).toBe(all.length);
  });

  test("expands a selection of groups into their raw statuses", () => {
    expect(expandDeploymentStatusGroups(["building", "cancelled"])).toEqual([
      "starting",
      "building",
      "deploying",
      "network",
      "finalizing",
      "cancelled",
      "skipped",
    ]);
    expect(expandDeploymentStatusGroups([])).toEqual([]);
  });

  test("group names follow declaration order", () => {
    expect(DEPLOYMENT_STATUS_GROUP_NAMES).toEqual([
      "ready",
      "failed",
      "building",
      "queued",
      "blocked",
      "cancelled",
      "superseded",
      "stopped",
    ]);
  });
});
