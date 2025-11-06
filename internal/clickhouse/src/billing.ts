import { z } from "zod";
import type { Querier } from "./client";

// get the billable ratelimits for a workspace in a specific month.
// month is not zero-indexed -> January = 1
export function getBillableRatelimits(ch: Querier) {
  return async (args: {
    workspaceId: string;
    year: number;
    month: number;
  }): Promise<number> => {
    const query = ch.query({
      query: `
    SELECT
      sum(count) as count
    FROM default.billable_ratelimits_per_month_v2
    WHERE workspace_id = {workspaceId: String}
    AND year = {year: Int64}
    AND month = {month: Int64}
    GROUP BY workspace_id, year, month
    `,
      params: z.object({
        workspaceId: z.string(),
        year: z.number().int(),
        month: z.number().int().min(1).max(12),
      }),
      schema: z.object({
        count: z.number().int(),
      }),
    });

    const res = await query(args);
    if (res.err || res.val.length === 0) {
      return 0;
    }
    return res.val.at(0)?.count ?? 0;
  };
}
// get the billable verifications for a workspace in a specific month.
// month is not zero-indexed -> January = 1
export function getBillableVerifications(ch: Querier) {
  return async (args: {
    workspaceId: string;
    year: number;
    month: number;
  }): Promise<number> => {
    const query = ch.query({
      query: `
    SELECT
      sum(count) as count
    FROM billable_verifications_per_month_v2
    WHERE workspace_id = {workspaceId: String}
    AND year = {year: Int64}
    AND month = {month: Int64}
    GROUP BY workspace_id, year, month
    `,
      params: z.object({
        workspaceId: z.string(),
        year: z.number().int(),
        month: z.number().int().min(1).max(12),
      }),
      schema: z.object({
        count: z.number().int(),
      }),
    });

    const res = await query(args);
    if (res.err || res.val.length === 0) {
      return 0;
    }
    return res.val.at(0)?.count ?? 0;
  };
}
