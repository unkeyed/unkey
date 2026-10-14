import { Ok, type Result } from "@unkey/error";
import { describe, expect, it } from "vitest";
import type { z } from "zod";
import { type AuditLogsRequest, getAuditLogs } from "./audit-logs";
import type { QueryError } from "./client/error";
import type { Querier } from "./client/interface";
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

function queryBy(queries: string[], needle: string): string {
  const query = queries.find((candidate) => candidate.includes(needle));
  if (!query) {
    throw new Error(`query containing ${needle} missing`);
  }
  return query;
}

class ScriptedQuerier implements Querier {
  public readonly queries: string[] = [];
  public readonly params: unknown[] = [];
  private readonly rowsByQuery: unknown[][];

  constructor(rowsByQuery: unknown[][]) {
    this.rowsByQuery = rowsByQuery;
  }

  public query<TIn extends z.ZodType<unknown>, TOut extends z.ZodType<unknown>>(req: {
    query: string;
    params?: TIn;
    schema: TOut;
  }): (params: z.input<TIn>) => Promise<Result<z.output<TOut>[], QueryError>> {
    const index = this.queries.length;
    this.queries.push(req.query);
    return async (params) => {
      this.params.push(params);
      return Ok((this.rowsByQuery[index] ?? []) as z.output<TOut>[]);
    };
  }
}

describe("getAuditLogs", () => {
  // The ordered scan has to stay on the primary key and off native JSON.
  // Sorting the retention window, or decoding JSON while scanning it, exceeds
  // the dashboard timeout once a workspace holds hundreds of billions of events.
  it("selects the page in primary-key order and decodes JSON by those keys", () => {
    const ch = new CapturingQuerier();

    getAuditLogs(ch)(baseRequest);

    const page = queryBy(ch.queries, "ORDER BY time DESC, event_id DESC");
    expect(page).toContain("SETTINGS optimize_read_in_order = 1");
    expect(page).toContain("LIMIT {limit: Int} OFFSET {offset: Int}");
    expect(page).toContain("time BETWEEN {startTime: UInt64} AND {endTime: UInt64}");
    expect(page).not.toContain("toJSONString");
    expect(page).not.toContain("actor_meta");
    expect(page).not.toContain("targets.meta");

    const json = queryBy(ch.queries, "toJSONString");
    expect(json).toContain("time IN {times: Array(Int64)}");
    expect(json).toContain("arrayZip({times: Array(Int64)}, {eventIds: Array(String)})");
    expect(json).not.toContain("BETWEEN");
    expect(json).not.toContain("ORDER BY");
    expect(json).not.toContain("OFFSET");

    const count = queryBy(ch.queries, "count()");
    expect(count).not.toContain("toJSONString");
  });

  it("adds event and actor predicates only to the window scans", () => {
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
    const page = queryBy(filtered.queries, "ORDER BY time DESC");
    const count = queryBy(filtered.queries, "count()");
    for (const query of [page, count]) {
      expect(query).toContain("event IN {events: Array(String)}");
      expect(query).toContain("actor_id IN {actorIds: Array(String)}");
    }
    const json = queryBy(filtered.queries, "toJSONString");
    expect(json).not.toContain("event IN");
    expect(json).not.toContain("actor_id IN");
  });

  it("keeps page order when attaching JSON for those keys", async () => {
    const ch = new ScriptedQuerier([
      [
        {
          eventId: "evt_newer",
          time: 20,
          event: "key.create",
          description: "newer",
          actorType: "user",
          actorId: "user_1",
          actorName: "Ada",
          remoteIp: "127.0.0.1",
          userAgent: "test",
        },
        {
          eventId: "evt_older",
          time: 10,
          event: "key.delete",
          description: "older",
          actorType: "key",
          actorId: "key_1",
          actorName: "",
          remoteIp: "",
          userAgent: "",
        },
      ],
      [
        {
          eventId: "evt_older",
          time: 10,
          actorMeta: "{}",
          meta: '{"id":10}',
          targets: [["key", "key_1", "", "{}"]],
        },
        {
          eventId: "evt_newer",
          time: 20,
          actorMeta: '{"role":"admin"}',
          meta: "{}",
          targets: [],
        },
      ],
    ]);

    const result = await getAuditLogs(ch)(baseRequest).getLogsQuery();

    expect(result.err).toBeUndefined();
    expect(result.val?.map((row) => row.eventId)).toEqual(["evt_newer", "evt_older"]);
    expect(result.val?.[0]).toMatchObject({
      description: "newer",
      actorMeta: '{"role":"admin"}',
      meta: "{}",
      targets: [],
    });
    expect(result.val?.[1]?.targets).toEqual([["key", "key_1", "", "{}"]]);
    expect(ch.params[1]).toMatchObject({
      workspaceId: "ws_123",
      bucketId: "unkey_mutations",
      times: [20, 10],
      eventIds: ["evt_newer", "evt_older"],
    });
  });

  it("does not look up JSON for an empty page", async () => {
    const ch = new ScriptedQuerier([[]]);

    const result = await getAuditLogs(ch)(baseRequest).getLogsQuery();

    expect(result.err).toBeUndefined();
    expect(result.val).toEqual([]);
    expect(ch.params).toHaveLength(1);
  });
});
