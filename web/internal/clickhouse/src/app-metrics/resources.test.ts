import { describe, expect, it } from "vitest";
import { CapturingQuerier } from "../test-utils";
import { DAY_MS, clampedGrid, latestGrid } from "./grid";
import { getAppResourceHistory, getAppResources, getAppResourcesByInstance } from "./resources";

const now = Date.parse("2026-10-19T12:45:20Z");
const scope = { workspaceId: "ws_acme", environmentId: "env_production" };

describe("getAppResources", () => {
  it.each([
    ["1h", "instance_resources_per_minute_v1"],
    ["24h", "instance_resources_per_minute_v1"],
    ["7d", "instance_resources_per_hour_v1"],
    ["30d", "instance_resources_per_hour_v1"],
  ] as const)("reads %s from %s", async (range, table) => {
    const ch = new CapturingQuerier();
    await getAppResources(ch)({ ...scope, grid: latestGrid(range, now) }, now);
    expect(ch.params[0]).toMatchObject({ table });
  });

  it("reads one bucket before the range start and binds the grid", async () => {
    const ch = new CapturingQuerier();
    await getAppResources(ch)({ ...scope, grid: latestGrid("1h", now) }, now);
    expect(ch.params[0]).toEqual({
      workspaceId: "ws_acme",
      environmentId: "env_production",
      table: "instance_resources_per_minute_v1",
      gaugeTable: "instance_resources_per_minute_v1",
      sourceSec: 60,
      bucketSec: 60,
      readStartSec: Date.parse("2026-10-19T11:44:00Z") / 1000,
      startSec: Date.parse("2026-10-19T11:45:00Z") / 1000,
      endSec: Date.parse("2026-10-19T12:45:00Z") / 1000,
      startMs: Date.parse("2026-10-19T11:45:00Z"),
      endMs: Date.parse("2026-10-19T12:45:00Z"),
      bucketMs: 60_000,
    });
  });

  it("chains counter deltas only across directly adjacent buckets", async () => {
    const ch = new CapturingQuerier();
    await getAppResources(ch)({ ...scope, grid: latestGrid("1h", now) }, now);
    expect(ch.queries[0]).toContain("if(b - lagInFrame(b) OVER w = {bucketSec: UInt32}");
    expect(ch.queries[0]).not.toContain("maxGapSec");
  });

  it("averages the app's rates over the whole bucket so overlapping containers add up", async () => {
    const ch = new CapturingQuerier();
    await getAppResources(ch)({ ...scope, grid: latestGrid("1h", now) }, now);
    const sql = ch.queries[0].replace(/\s+/g, " ");
    expect(sql).toContain("toNullable(sum(cpu_delta) / {bucketSec: UInt32} / 1000) AS cpu");
    expect(sql).toContain("toNullable(sum(ingress_delta) / {bucketSec: UInt32}) AS ingress");
  });

  it("takes the app's gauges from the highest per-minute concurrent total", async () => {
    const ch = new CapturingQuerier();
    await getAppResources(ch)({ ...scope, grid: latestGrid("30d", now) }, now);
    const sql = ch.queries[0].replace(/\s+/g, " ");
    expect(sql).toContain("FROM {gaugeTable: Identifier}");
    expect(sql).toContain("GROUP BY b, time, instance_id");
    expect(sql).toContain("sum(memory) AS memory, sum(disk) AS disk, count() AS instances");
    expect(sql).toContain("toNullable(max(memory)) AS memory");
    expect(sql).not.toContain("toFloat64(sum(memory))");
  });

  it.each([
    ["1h", 0, "instance_resources_per_minute_v1"],
    ["30d", 0, "instance_resources_per_minute_v1"],
    ["7d", 60 * DAY_MS, "instance_resources_per_hour_v1"],
  ] as const)("reads %s gauges ending %d ms ago from %s", async (range, agoMs, gaugeTable) => {
    const ch = new CapturingQuerier();
    await getAppResources(ch)({ ...scope, grid: clampedGrid(range, now - agoMs, now) }, now);
    expect(ch.params[0]).toMatchObject({ gaugeTable });
  });

  it("drops the leading bucket", async () => {
    const ch = new CapturingQuerier();
    await getAppResources(ch)({ ...scope, grid: latestGrid("1h", now) }, now);
    expect(ch.queries[0]).toContain("WHERE b >= {startSec: UInt32}");
    expect(ch.queries[0]).toContain("GROUP BY t\n");
  });

  it("reads the environment's deployment ids from the day rollup, without app_id", async () => {
    const ch = new CapturingQuerier();
    await getAppResources(ch)({ ...scope, grid: latestGrid("1h", now) }, now);
    const sql = ch.queries[0].replace(/\s+/g, " ");
    expect(sql).toContain(
      "AND resource_id IN ( SELECT DISTINCT resource_id FROM instance_resources_per_day_v1 WHERE workspace_id = {workspaceId: String} AND resource_type = 'deployment' AND environment_id = {environmentId: String} AND time >= toStartOfDay(toDateTime({readStartSec: UInt32})) AND time < toDateTime({endSec: UInt32}) )",
    );
    expect(sql).not.toContain("app_id");
    expect(sql).not.toContain("deploymentIds");
  });
});

describe("getAppResources gauges at a bucket boundary", () => {
  it("keeps 30d on the minute rollup in the seconds before a bucket settles", async () => {
    const justAfterBoundary = Date.parse("2026-10-19T18:00:10Z");
    const ch = new CapturingQuerier();
    await getAppResources(ch)(
      { ...scope, grid: latestGrid("30d", justAfterBoundary) },
      justAfterBoundary,
    );
    expect(ch.params[0]).toMatchObject({
      endMs: Date.parse("2026-10-19T12:00:00Z"),
      gaugeTable: "instance_resources_per_minute_v1",
    });
  });
});

describe("getAppResourcesByInstance", () => {
  const instances = ["kebap-7c4d2e9a1-x8k2p", "kebap-7c4d2e9a1-q3m9z"];

  it("reads only the named pods and drops the fill series of an empty result", async () => {
    const ch = new CapturingQuerier();
    await getAppResourcesByInstance(ch)({ ...scope, instances, grid: latestGrid("24h", now) }, now);
    expect(ch.params[0]).toMatchObject({ instances });
    expect(ch.queries[0]).toContain("AND instance_id IN {instances: Array(String)}");
    expect(ch.queries[0]).toContain("WHERE instance != ''");
  });

  it("averages a pod over the seconds its containers ran and keeps its peak memory", async () => {
    const ch = new CapturingQuerier();
    await getAppResourcesByInstance(ch)({ ...scope, instances, grid: latestGrid("7d", now) }, now);
    const sql = ch.queries[0].replace(/\s+/g, " ");
    expect(ch.params[0]).toMatchObject({ sourceSec: 3600 });
    expect(sql).toContain(
      "toUnixTimestamp(last_time) - toUnixTimestamp(first_time) + {sourceSec: UInt32} AS covered_sec",
    );
    expect(sql).toContain(
      "toNullable(sum(cpu_delta) / least(sum(covered_sec), {bucketSec: UInt32}) / 1000) AS cpu",
    );
    expect(sql).toContain("max(memory) AS memory, max(disk) AS disk");
    expect(sql).toContain("GROUP BY instance, t ) GROUP BY instance, t");
  });

  it("groups by pod name", async () => {
    const ch = new CapturingQuerier();
    await getAppResourcesByInstance(ch)({ ...scope, instances, grid: latestGrid("6h", now) }, now);
    expect(ch.queries[0]).toContain("GROUP BY instance, t");
    expect(ch.params[0]).toMatchObject({
      table: "instance_resources_per_minute_v1",
      bucketSec: 300,
    });
  });
});

describe("getAppResourceHistory", () => {
  it("passes the hourly source width and grid end that cap lastSeenMs", async () => {
    const ch = new CapturingQuerier();
    await getAppResourceHistory(ch)({ ...scope, instances: [], grid: latestGrid("7d", now) }, now);
    expect(ch.params[0]).toMatchObject({
      sourceSec: 3600,
      endMs: Date.parse("2026-10-19T12:00:00Z"),
    });
    expect(ch.queries[0].replace(/\s+/g, " ")).toContain(
      "least(observed.bucketMs + toInt64({sourceSec: UInt32}) * 1000, {endMs: Int64}) AS lastSeenMs",
    );
  });
});
