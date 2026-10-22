import { describe, expect, it } from "vitest";
import { CapturingQuerier } from "../test-utils";
import { latestGrid } from "./grid";
import { getAppResources, getAppResourcesByInstance } from "./resources";

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

  it("averages the app over the whole bucket so overlapping containers add up", async () => {
    const ch = new CapturingQuerier();
    await getAppResources(ch)({ ...scope, grid: latestGrid("1h", now) }, now);
    const sql = ch.queries[0].replace(/\s+/g, " ");
    expect(sql).toContain("toNullable(sum(cpu_delta) / {bucketSec: UInt32} / 1000) AS cpu");
    expect(sql).toContain("max(memory) AS memory, max(disk) AS disk");
    expect(sql).toContain("GROUP BY instance, t ) GROUP BY t");
    expect(sql).toContain("toNullable(toFloat64(sum(memory))) AS memory");
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

describe("getAppResourcesByInstance", () => {
  it("averages a pod over the seconds its containers ran and keeps its peak memory", async () => {
    const ch = new CapturingQuerier();
    await getAppResourcesByInstance(ch)({ ...scope, grid: latestGrid("7d", now) }, now);
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
    await getAppResourcesByInstance(ch)({ ...scope, grid: latestGrid("6h", now) }, now);
    expect(ch.queries[0]).toContain("GROUP BY instance, t");
    expect(ch.params[0]).toMatchObject({
      table: "instance_resources_per_minute_v1",
      bucketSec: 300,
    });
  });
});
