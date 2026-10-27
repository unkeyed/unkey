import type { InstanceEvent } from "@/lib/metrics/app-metrics";
import { describe, expect, it } from "vitest";
import { crashEpisodes, episodeLabel, episodeMarkers, eventsInScope } from "./events-model";

function event(id: string, overrides: Partial<InstanceEvent> = {}): InstanceEvent {
  return {
    id,
    atMs: 0,
    instance: "us-0",
    instanceId: "ins_us0",
    region: "us-east-1",
    deploymentId: "d_1",
    kind: "crashed",
    reason: "Error",
    exitCode: 1,
    ...overrides,
  };
}

describe("eventsInScope", () => {
  const events = [
    event("old-us", { atMs: 1_000 }),
    event("new-eu", { atMs: 5_000, instance: "eu-0", region: "eu-central-1" }),
    event("mid-us1", { atMs: 3_000, instance: "us-1" }),
  ];
  const ids = (selection: { regions: string[]; instances: string[] }) =>
    eventsInScope(events, selection).map((e) => e.id);

  it("keeps every event when nothing is selected", () => {
    expect(ids({ regions: [], instances: [] })).toEqual(["old-us", "new-eu", "mid-us1"]);
  });

  it("narrows to the selected regions and instances", () => {
    expect(ids({ regions: ["us-east-1"], instances: [] })).toEqual(["old-us", "mid-us1"]);
    expect(ids({ regions: ["us-east-1"], instances: ["us-1"] })).toEqual(["mid-us1"]);
  });
});

describe("crashEpisodes", () => {
  const MIN = 60_000;
  const END = 24 * 60 * MIN;
  const summary = (events: InstanceEvent[], gridEndMs = END) =>
    crashEpisodes(events, gridEndMs).map((e) => [
      e.instance,
      e.kind,
      e.startMs,
      e.endMs,
      e.restarts,
    ]);

  it("collapses a backoff sequence into one episode that ends when the pod recovers", () => {
    expect(
      summary([
        event("c1", { atMs: 0, kind: "oom", exitCode: 137 }),
        event("w1", { atMs: 1_000, kind: "crashloop", exitCode: null }),
        event("s1", { atMs: 20_000, kind: "started", exitCode: null }),
        event("c2", { atMs: 1 * MIN, kind: "crashed" }),
        event("w2", { atMs: 1 * MIN + 1_000, kind: "crashloop", exitCode: null }),
        event("s2", { atMs: 3 * MIN, kind: "started", exitCode: null }),
        event("c3", { atMs: 8 * MIN, kind: "oom", exitCode: 137 }),
        event("s3", { atMs: 13 * MIN, kind: "started", exitCode: null }),
        event("later", { atMs: 60 * MIN, kind: "started", exitCode: null }),
      ]),
    ).toEqual([["us-0", "oom", 0, 13 * MIN, 3]]);
  });

  it("keeps crash runs 30 minutes apart as two episodes", () => {
    expect(
      summary([
        event("a", { atMs: 0 }),
        event("b", { atMs: 2 * MIN }),
        event("c", { atMs: 32 * MIN }),
      ]),
    ).toEqual([
      ["us-0", "crashed", 0, 2 * MIN, 2],
      ["us-0", "crashed", 32 * MIN, 32 * MIN, 1],
    ]);
  });

  it("never merges different instances", () => {
    expect(
      summary([
        event("a", { atMs: 0 }),
        event("b", { atMs: 1_000, instance: "us-1", kind: "oom" }),
      ]),
    ).toEqual([
      ["us-0", "crashed", 0, 0, 1],
      ["us-1", "oom", 1_000, 1_000, 1],
    ]);
  });

  it("never merges the same pod name in different regions", () => {
    expect(
      summary([event("a", { atMs: 0 }), event("b", { atMs: 1_000, region: "eu-central-1" })]).map(
        ([instance, , startMs]) => [instance, startMs],
      ),
    ).toEqual([
      ["us-0", 0],
      ["us-0", 1_000],
    ]);
  });

  it("runs an unrecovered episode to the grid end while the pod still crashes", () => {
    expect(summary([event("a", { atMs: 0 }), event("b", { atMs: 2 * MIN })], 5 * MIN)).toEqual([
      ["us-0", "crashed", 0, 5 * MIN, 2],
    ]);
  });

  it("ends an unrecovered episode at its last event once it has gone quiet", () => {
    expect(summary([event("a", { atMs: 0 }), event("b", { atMs: 2 * MIN })], 30 * MIN)).toEqual([
      ["us-0", "crashed", 0, 2 * MIN, 2],
    ]);
  });

  it("counts a single crash as one restart and marks it as a diamond", () => {
    const episodes = crashEpisodes([event("solo", { atMs: 5_000 })], END);
    expect(episodeMarkers(episodes)).toEqual([
      {
        id: "episode:us-0::us-east-1:5000",
        atMs: 5_000,
        span: { endMs: 5_000 },
        shape: "diamond",
        label: "Crashed",
      },
    ]);
  });

  it("labels a looping episode with its restart count", () => {
    const [episode] = crashEpisodes(
      [event("a", { atMs: 0, kind: "oom" }), event("b", { atMs: MIN })],
      END,
    );
    expect(episodeLabel(episode)).toBe("Out of memory, 2 restarts");
  });
});
