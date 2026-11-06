import { describe, expect, it } from "vitest";
import { type RatelimitOverviewLogsParams, getRatelimitOverviewLogs } from "./ratelimits";
import { CapturingQuerier } from "./test-utils";

const HOUR_MS = 60 * 60 * 1000;
const DAY_MS = 24 * HOUR_MS;
const START_TIME = 1_700_000_000_000;

function baseArgs(
  overrides: Partial<RatelimitOverviewLogsParams> = {},
): RatelimitOverviewLogsParams {
  return {
    workspaceId: "ws_123",
    namespaceId: "rlns_123",
    limit: 50,
    startTime: START_TIME,
    endTime: START_TIME + 12 * HOUR_MS,
    status: null,
    identifiers: null,
    page: 1,
    sorts: null,
    ...overrides,
  };
}

async function runQuery(args: RatelimitOverviewLogsParams): Promise<string> {
  const ch = new CapturingQuerier();
  const { logsQuery, countQuery } = getRatelimitOverviewLogs(ch)(args);
  await Promise.all([logsQuery, countQuery]);
  const logs = ch.queries.find((query) => query.includes("aggregated_data"));
  if (!logs) {
    throw new Error("overview logs query missing");
  }
  return logs;
}

describe("getRatelimitOverviewLogs", () => {
  it("aggregates from the per-minute rollup instead of the raw decision stream", async () => {
    const query = await runQuery(baseArgs());

    expect(query).toContain("default.ratelimits_per_minute_v2");
    expect(query).not.toContain("ratelimits_raw_v2");
  });

  it("does not select the per-request request_id", async () => {
    const query = await runQuery(baseArgs());

    expect(query).not.toContain("request_id");
    expect(query).toContain("GROUP BY identifier");
  });

  it("picks the coarsest rollup whose retention covers the window", async () => {
    const hourly = await runQuery(baseArgs({ endTime: START_TIME + 20 * DAY_MS }));
    expect(hourly).toContain("default.ratelimits_per_hour_v2");

    const daily = await runQuery(baseArgs({ endTime: START_TIME + 60 * DAY_MS }));
    expect(daily).toContain("default.ratelimits_per_day_v2");

    const monthly = await runQuery(baseArgs({ endTime: START_TIME + 200 * DAY_MS }));
    expect(monthly).toContain("default.ratelimits_per_month_v2");
  });

  it("narrows the aggregated rows to a single selected status", async () => {
    const passed = await runQuery(baseArgs({ status: [{ operator: "is", value: "passed" }] }));
    expect(passed).toContain("HAVING sum(r.passed) > 0");
    expect(passed).toContain("0 as blocked_count");

    const blocked = await runQuery(baseArgs({ status: [{ operator: "is", value: "blocked" }] }));
    expect(blocked).toContain("HAVING sum(r.total) - sum(r.passed) > 0");
    expect(blocked).toContain("0 as passed_count");
  });

  it("keeps the normal aggregate when both statuses are selected", async () => {
    const both = await runQuery(
      baseArgs({
        status: [
          { operator: "is", value: "passed" },
          { operator: "is", value: "blocked" },
        ],
      }),
    );
    expect(both).not.toContain("HAVING");
    expect(both).toContain("sum(r.passed) as passed_count");
  });
});
