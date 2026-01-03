import { z } from "zod";
import type { Querier } from "./client";
import { dateTimeToUnix } from "./util";

// get the billable verifications for a workspace in a specific month.
// month is not zero-indexed -> January = 1
export function getActiveWorkspacesPerMonth(ch: Querier) {
  return async () => {
    const query = ch.query({
      query: `
    SELECT
      count(DISTINCT workspace_id) as workspaces,
      time
    FROM default.active_workspaces_per_month_v2
    GROUP BY time
    ORDER BY time ASC
    ;`,
      schema: z.object({
        time: dateTimeToUnix,
        workspaces: z.number().int(),
      }),
    });

    return await query({});
  };
}
