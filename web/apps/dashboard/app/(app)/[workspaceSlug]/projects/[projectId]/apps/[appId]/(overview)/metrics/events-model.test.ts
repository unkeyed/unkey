import { formatSpan } from "@/components/charts/format-interval";
import type { InstanceEvent } from "@/lib/metrics/app-metrics";
import { describe, expect, it } from "vitest";
import {
  type CrashEpisode,
  crashEpisodes,
  crashIncidents,
  episodeError,
  episodeGroup,
  episodeLabel,
  eventsInScope,
  incidentOf,
  logsWindow,
  spanEnd,
} from "./events-model";

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
    message: null,
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
        event("s", { atMs: 33 * MIN, kind: "started", exitCode: null }),
      ]),
    ).toEqual([
      ["us-0", "crashed", 0, 2 * MIN, 2],
      ["us-0", "crashed", 32 * MIN, 33 * MIN, 1],
    ]);
  });

  it("never merges different instances", () => {
    expect(
      summary([
        event("a", { atMs: 0, kind: "oom" }),
        event("b", { atMs: 1_000, instance: "us-1", kind: "oom" }),
      ]),
    ).toEqual([
      ["us-0", "oom", 0, 0, 1],
      ["us-1", "oom", 1_000, 1_000, 1],
    ]);
  });

  it("never merges the same pod name in different regions", () => {
    expect(
      summary([
        event("a", { atMs: 0, kind: "oom" }),
        event("b", { atMs: 1_000, kind: "oom", region: "eu-central-1" }),
      ]).map(([instance, , startMs]) => [instance, startMs]),
    ).toEqual([
      ["us-0", 0],
      ["us-0", 1_000],
    ]);
  });

  it("runs an unrecovered episode to the grid end while the pod still crashes", () => {
    const looping = [
      event("a", { atMs: 0 }),
      event("b", { atMs: 2 * MIN }),
      event("w", { atMs: 2 * MIN + 1_000, kind: "crashloop", exitCode: null }),
    ];
    expect(summary(looping, 5 * MIN)).toEqual([["us-0", "crashed", 0, 5 * MIN, 2]]);
    expect(crashEpisodes(looping, 5 * MIN).map((e) => e.ongoing)).toEqual([true]);
  });

  it("ends an unrecovered episode at its last event once it has gone quiet", () => {
    const looping = [
      event("a", { atMs: 0 }),
      event("b", { atMs: 2 * MIN }),
      event("w", { atMs: 2 * MIN + 1_000, kind: "crashloop", exitCode: null }),
    ];
    expect(summary(looping, 30 * MIN)).toEqual([["us-0", "crashed", 0, 2 * MIN + 1_000, 2]]);
    expect(crashEpisodes(looping, 30 * MIN).map((e) => e.ongoing)).toEqual([false]);
  });

  it("counts a single crash as one restart", () => {
    const episodes = crashEpisodes([event("solo", { atMs: 5_000, kind: "oom" })], END);
    expect(episodes.map((e) => [e.id, e.startMs, e.endMs, e.restarts, episodeLabel(e)])).toEqual([
      ["us-0::us-east-1:5000", 5_000, 5_000, 1, "Out of memory"],
    ]);
  });

  it("labels a looping episode with its restart count", () => {
    const [episode] = crashEpisodes(
      [
        event("a", { atMs: 0, kind: "oom" }),
        event("b", { atMs: MIN }),
        event("s", { atMs: 2 * MIN, kind: "started", exitCode: null }),
      ],
      END,
    );
    expect(episodeLabel(episode)).toBe("Out of memory, 2 restarts");
  });

  it("drops a pod that stopped and never came back, since that was a shutdown", () => {
    expect(
      summary([
        event("s", { atMs: 0, kind: "started", exitCode: null }),
        event("stop", { atMs: 30 * MIN, exitCode: 143 }),
      ]),
    ).toEqual([]);
  });

  it("keeps a crash that has not restarted yet when it just happened", () => {
    expect(summary([event("fresh", { atMs: END - MIN })])).toEqual([
      ["us-0", "crashed", END - MIN, END, 1],
    ]);
  });

  it("keeps an out-of-memory kill even when the pod never came back", () => {
    expect(summary([event("oom", { atMs: 0, kind: "oom", exitCode: 137 })])).toEqual([
      ["us-0", "oom", 0, 0, 1],
    ]);
  });

  it("takes the error message only from a terminated event, never from a waiting one", () => {
    const backoff = (instance: string) =>
      event(`w-${instance}`, {
        instance,
        kind: "crashloop",
        exitCode: null,
        reason: "CrashLoopBackOff",
        message: `back-off 5m0s restarting failed container=app pod=${instance}`,
      });
    const episodes = crashEpisodes(
      [
        event("c", { atMs: 0, message: "Error: boom" }),
        event("w", { atMs: 1_000, kind: "crashloop", exitCode: null, message: "back-off" }),
        backoff("us-1"),
        backoff("us-2"),
      ],
      END,
    );
    expect(episodes.map((e) => [e.instance, e.exitCode, e.reason, e.message])).toEqual([
      ["us-0", 1, "Error", "Error: boom"],
      ["us-1", null, "CrashLoopBackOff", null],
      ["us-2", null, "CrashLoopBackOff", null],
    ]);
    expect(episodeGroup(episodes.slice(1)).errors.map((e) => [e.title, e.count])).toEqual([
      ["Crashing", 2],
    ]);
  });
});

describe("logsWindow", () => {
  it("opens logs from 15 minutes before an episode to 5 minutes after it", () => {
    const [episode] = crashEpisodes(
      [
        event("a", { atMs: 60 * 60_000, kind: "oom" }),
        event("s", { atMs: 62 * 60_000, kind: "started", exitCode: null }),
      ],
      24 * 60 * 60_000,
    );
    expect(logsWindow(episode)).toEqual({ startMs: 45 * 60_000, endMs: 67 * 60_000 });
  });
});

describe("episodeGroup", () => {
  const MIN = 60_000;
  const episode = (instance: string, startMs: number, overrides: Partial<CrashEpisode> = {}) => ({
    id: `${instance}:${startMs}`,
    instance,
    instanceId: `ins_${instance}`,
    region: "us-east-1",
    deploymentId: "d_1",
    kind: "crashed" as const,
    startMs,
    endMs: startMs + MIN,
    ongoing: false,
    latestMs: startMs,
    restarts: 1,
    exitCode: 1,
    reason: "Error",
    message: null as string | null,
    ...overrides,
  });

  it("orders episodes oldest first and spans the logs window", () => {
    const group = episodeGroup([
      episode("us-2", 300 * MIN),
      episode("us-0", 100 * MIN, { kind: "oom", endMs: 400 * MIN }),
      episode("us-1", 200 * MIN),
      episode("us-1", 250 * MIN),
    ]);
    expect(group.span).toEqual({ startMs: 100 * MIN, endMs: 400 * MIN });
    expect(group.episodes.map((e) => e.id)).toEqual([
      `us-0:${100 * MIN}`,
      `us-1:${200 * MIN}`,
      `us-1:${250 * MIN}`,
      `us-2:${300 * MIN}`,
    ]);
    expect(group.logs).toEqual({
      deploymentIds: ["d_1"],
      instanceIds: ["ins_us-0", "ins_us-1", "ins_us-2"],
      window: { startMs: 85 * MIN, endMs: 405 * MIN },
    });
  });

  it("groups episodes by the first line of their error, most frequent first", () => {
    const redis =
      "Error: connect ECONNREFUSED 10.43.12.7:6379\n    at TCPConnectWrap.afterConnect\n";
    const oom = "Container app exceeded its memory limit of 512Mi and was OOM killed";
    const group = episodeGroup([
      episode("a", 0, { kind: "oom", message: oom, exitCode: 137, reason: "OOMKilled" }),
      episode("b", MIN, { message: redis, region: "eu-central-1", deploymentId: "d_2" }),
      episode("c", 2 * MIN, { message: redis }),
      episode("d", 3 * MIN, { message: `  ${redis.trimEnd()}\n    at other` }),
    ]);
    expect(group.errors).toEqual([
      {
        title: "Error: connect ECONNREFUSED 10.43.12.7:6379",
        detail: "at TCPConnectWrap.afterConnect",
        count: 3,
        kinds: ["crashed"],
        regions: ["eu-central-1", "us-east-1"],
        logs: {
          deploymentIds: ["d_2", "d_1"],
          instanceIds: ["ins_b", "ins_c", "ins_d"],
          window: { startMs: -14 * MIN, endMs: 9 * MIN },
        },
      },
      {
        title: oom,
        detail: null,
        count: 1,
        kinds: ["oom"],
        regions: ["us-east-1"],
        logs: {
          deploymentIds: ["d_1"],
          instanceIds: ["ins_a"],
          window: { startMs: -15 * MIN, endMs: 6 * MIN },
        },
      },
    ]);
  });

  it("breaks count ties by first occurrence", () => {
    const group = episodeGroup([
      episode("b", MIN, { message: "second" }),
      episode("a", 0, { message: "first" }),
    ]);
    expect(group.errors.map((e) => e.title)).toEqual(["first", "second"]);
  });

  it("names an error with no message after its exit", () => {
    const group = episodeGroup([
      episode("a", 0),
      episode("b", MIN),
      episode("c", 2 * MIN, { kind: "oom", exitCode: 137, reason: "OOMKilled" }),
      episode("d", 3 * MIN, { kind: "crashloop", exitCode: null, reason: "CrashLoopBackOff" }),
    ]);
    expect(group.errors.map((e) => [e.title, e.count, e.detail])).toEqual([
      ["Exited with error · exit=1", 2, null],
      ["Out of memory · exit=137", 1, null],
      ["Crashing", 1, null],
    ]);
  });

  it("keeps every deployment when episodes span deployments", () => {
    const group = episodeGroup([
      episode("a", 0, { deploymentId: "d_1" }),
      episode("a", 20 * MIN, { deploymentId: "d_2" }),
    ]);
    expect([group.logs.deploymentIds, group.logs.instanceIds]).toEqual([["d_1", "d_2"], ["ins_a"]]);
  });

  it("drops the instance filter when a crashed pod has no instance id", () => {
    const group = episodeGroup([episode("a", 0), episode("b", MIN, { instanceId: null })]);
    expect(group.logs.instanceIds).toEqual([]);
  });
});

describe("episodeError", () => {
  const MIN = 60_000;
  const [base] = crashEpisodes([event("c", { atMs: 0, exitCode: 137, reason: "OOMKilled" })], MIN);

  it("names the exit when the error carries a message", () => {
    expect(episodeError({ ...base, message: "Error: boom\n  at main" })).toEqual({
      title: "Error: boom",
      detail: "at main",
      exit: { code: 137, reason: "OOMKilled" },
    });
  });

  it("puts the exit in the title when there is no message", () => {
    expect(episodeError(base)).toEqual({
      title: "Out of memory · exit=137",
      detail: null,
      exit: null,
    });
  });
});

describe("crashIncidents", () => {
  const MIN = 60_000;
  const episode = (instance: string, startMs: number): CrashEpisode => ({
    id: `${instance}:${startMs}`,
    instance,
    instanceId: `ins_${instance}`,
    region: "us-east-1",
    deploymentId: "d_1",
    kind: "crashed",
    startMs,
    endMs: startMs + MIN,
    ongoing: false,
    latestMs: startMs,
    restarts: 1,
    exitCode: 1,
    reason: "Error",
    message: null,
  });

  it("chains episodes across instances while each starts within 15 minutes of the run's end", () => {
    const incidents = crashIncidents([
      episode("b", 490 * MIN),
      episode("a", 480 * MIN),
      episode("c", 505 * MIN),
      episode("a", 522 * MIN),
      episode("b", 630 * MIN),
    ]);
    expect(incidents.map((i) => [i.span, i.episodes.length])).toEqual([
      [{ startMs: 480 * MIN, endMs: 506 * MIN }, 3],
      [{ startMs: 522 * MIN, endMs: 523 * MIN }, 1],
      [{ startMs: 630 * MIN, endMs: 631 * MIN }, 1],
    ]);
  });

  it("keeps a gap of exactly 15 minutes after the previous episode ends in the same incident", () => {
    const incidents = crashIncidents([episode("a", 0), episode("b", 16 * MIN)]);
    expect(incidents.map((i) => i.episodes.map((e) => e.id))).toEqual([["a:0", `b:${16 * MIN}`]]);
  });

  it("spans a long crash loop to its end and chains what starts soon after it", () => {
    const incidents = crashIncidents([
      { ...episode("a", 0), endMs: 120 * MIN },
      episode("b", 130 * MIN),
    ]);
    expect(incidents.map((i) => [i.span, i.episodes.length])).toEqual([
      [{ startMs: 0, endMs: 131 * MIN }, 2],
    ]);
  });

  it("returns no incidents without episodes", () => {
    expect(crashIncidents([])).toEqual([]);
  });
});

describe("incidentOf", () => {
  const MIN = 60_000;
  const at = (instance: string, atMs: number) =>
    event(`${instance}-${atMs}`, { instance, atMs, kind: "oom", exitCode: 137 });

  it("labels a lone episode by its kind and time", () => {
    const [group] = crashIncidents(crashEpisodes([at("us-0", 0)], 24 * 60 * MIN));
    expect(incidentOf(group)).toMatchObject({
      id: "incident:us-0::us-east-1:0",
      kind: "episode",
      label: `Out of memory, ${formatSpan(0, 0, true)}`,
      span: { startMs: 0, endMs: 0 },
    });
  });

  it("labels a group by its count and the range to the last episode's end", () => {
    const [group] = crashIncidents(
      crashEpisodes([at("us-0", 0), at("us-1", 5 * MIN), at("us-0", 8 * MIN)], 24 * 60 * MIN),
    );
    expect(incidentOf(group)).toMatchObject({
      kind: "group",
      label: `2 crash events, ${formatSpan(0, 8 * MIN, true)}`,
      span: { startMs: 0, endMs: 8 * MIN },
    });
  });

  it("leaves a group open to now while any of its episodes is ongoing", () => {
    const start = new Date(2026, 0, 1, 8, 0).getTime();
    const [group] = crashIncidents(
      crashEpisodes([at("us-0", start), at("us-1", start + 5 * MIN)], start + 12 * MIN),
    );
    expect(spanEnd(group.episodes)).toBe("now");
    expect(incidentOf(group)).toMatchObject({
      kind: "group",
      label: "2 crash events, Jan 1, 8:00 AM – now",
      span: { startMs: start, endMs: start + 12 * MIN },
    });
  });

  it("closes a group once every episode has gone quiet", () => {
    const [group] = crashIncidents(crashEpisodes([at("us-0", 0), at("us-1", 5 * MIN)], 60 * MIN));
    expect(spanEnd(group.episodes)).toBe(5 * MIN);
  });
});
