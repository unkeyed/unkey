import { z } from "zod";
import { clickhouse } from "./client";

// get the billable verifications for a workspace in a specific month.
// month is not zero-indexed -> January = 1
export async function getBillableVerifications(args: {
  workspaceId: string;
  year: number;
  month: number;
}): Promise<number> {
  const query = clickhouse.query({
    query: `
    SELECT
      sum(count) as count
    FROM billing.billable_verifications_per_month_v1
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
  if (!res) {
    return 0;
  }
  return res.at(0)?.count ?? 0;
}
