import { describe, expect, it } from "vitest";
import { type AuditLogsRequest, getAuditLogs } from "./audit-logs";
import { CapturingQuerier } from "./test-utils";

const baseRequest: AuditLogsRequest = {
  workspaceId: "ws_123",
  bucketId: "unkey_mutations",
  limit: 50,
  offset: 0,
  startTime: 1,
  endTime: 2,
  events: [],
  actorIds: [],
};

function logsSql(queries: string[]): string {
  const logs = queries.find((query) => query.includes("toJSONString"));
  if (!logs) {
    throw new Error("logs query missing");
  }
  return logs;
}

describe("getAuditLogs", () => {
  // The rows query decodes native JSON. Without these settings ClickHouse
  // reads those columns for the full 90-day window and the dashboard's 20s
  // timeout returns 500. The cap must cover the handler's max page of 200,
  // and read-in-order must stay off so an unfiltered primary-key scan still
  // lazy-materializes.
  it("decodes JSON only after the page is selected", () => {
    const ch = new CapturingQuerier();

    getAuditLogs(ch)(baseRequest);

    const logs = logsSql(ch.queries);
    expect(logs).toContain("optimize_read_in_order = 0");
    expect(logs).toContain("query_plan_optimize_lazy_materialization = 1");
    expect(logs).toContain("query_plan_max_limit_for_lazy_materialization = 200");
    expect(logs).toContain("LIMIT {limit: Int} OFFSET {offset: Int}");

    const count = ch.queries.find((query) => query.includes("count()"));
    expect(count).toBeDefined();
    expect(count).not.toContain("toJSONString");
    expect(count).not.toContain("query_plan_optimize_lazy_materialization");
  });

  it("adds event and actor predicates only when filters are set", () => {
    const unfiltered = new CapturingQuerier();
    getAuditLogs(unfiltered)(baseRequest);
    for (const query of unfiltered.queries) {
      expect(query).not.toContain("event IN");
      expect(query).not.toContain("actor_id IN");
    }

    const filtered = new CapturingQuerier();
    getAuditLogs(filtered)({
      ...baseRequest,
      events: ["key.create"],
      actorIds: ["user_123"],
    });
    for (const query of filtered.queries) {
      expect(query).toContain("event IN {events: Array(String)}");
      expect(query).toContain("actor_id IN {actorIds: Array(String)}");
    }
  });
});
