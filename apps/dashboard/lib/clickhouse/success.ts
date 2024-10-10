import { z } from "zod";
import { clickhouse } from "./client";

// get the billable verifications for a workspace in a specific month.
// month is not zero-indexed -> January = 1
export async function getActiveWorkspacesPerMonth() {
  const query = clickhouse.query({
    query: `
    SELECT 
      count(DISTINCT workspace_id) as workspaces,      
      time
    FROM business.active_workspaces_per_month_v1
    GROUP BY time
    ORDER BY time ASC
    ;`,
    schema: z.object({
      time: z.string().transform((s) => new Date(s).getTime()),
      workspaces: z.number().int(),
    }),
  });

  return await query({});
}
