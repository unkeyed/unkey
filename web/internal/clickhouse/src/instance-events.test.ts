import { describe, expect, it } from "vitest";
import { type CrashTimelineRequest, getCrashTimeline } from "./instance-events";
import { CapturingQuerier } from "./test-utils";

const request: CrashTimelineRequest = {
  workspaceId: "ws_123",
  projectId: "proj_123",
  environmentId: "env_123",
  startTime: 1,
  endTime: 2,
  waitingReasons: ["CrashLoopBackOff"],
  perPodLimit: 300,
  limit: 5_000,
};

describe("getCrashTimeline", () => {
  it("caps rows per pod before the overall cap", async () => {
    const ch = new CapturingQuerier();

    await getCrashTimeline(ch)(request);

    const [query] = ch.queries;
    expect(query).toContain("LIMIT {perPodLimit: Int} BY pod_name, region");
    expect(query.indexOf("LIMIT {perPodLimit: Int} BY")).toBeLessThan(
      query.indexOf("LIMIT {limit: Int}"),
    );
    expect(ch.params).toEqual([request]);
  });

  it("keeps only the waiting reasons it is given", async () => {
    const ch = new CapturingQuerier();

    await getCrashTimeline(ch)(request);

    expect(ch.queries[0]).toContain(
      "(event_kind != 'waiting' OR reason IN {waitingReasons: Array(String)})",
    );
  });
});
