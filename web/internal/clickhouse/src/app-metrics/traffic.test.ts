import { describe, expect, it } from "vitest";
import { CapturingQuerier } from "../test-utils";
import { latestGrid } from "./grid";
import { getAppTraffic } from "./traffic";

const now = Date.parse("2026-10-19T12:45:20Z");
const scope = {
  workspaceId: "ws_acme",
  projectId: "proj_acme",
  appId: "app_checkout",
  environmentId: "env_production",
};

describe("getAppTraffic", () => {
  it.each([
    ["1h", "frontline_requests_per_minute_v1"],
    ["6h", "frontline_requests_per_5m_v1"],
    ["24h", "frontline_requests_per_15m_v1"],
    ["3d", "frontline_requests_per_hour_v1"],
    ["30d", "frontline_requests_per_hour_v1"],
  ] as const)("reads %s from %s", async (range, table) => {
    const ch = new CapturingQuerier();
    await getAppTraffic(ch)({ ...scope, grid: latestGrid(range, now) }, now);
    expect(ch.params[0]).toMatchObject({ table });
  });

  it("binds the grid without a leading bucket", async () => {
    const ch = new CapturingQuerier();
    await getAppTraffic(ch)({ ...scope, grid: latestGrid("24h", now) }, now);
    expect(ch.params[0]).toEqual({
      workspaceId: "ws_acme",
      projectId: "proj_acme",
      appId: "app_checkout",
      environmentId: "env_production",
      table: "frontline_requests_per_15m_v1",
      bucketSec: 900,
      startMs: Date.parse("2026-10-18T12:45:00Z"),
      endMs: Date.parse("2026-10-19T12:45:00Z"),
      bucketMs: 900_000,
    });
  });
});
