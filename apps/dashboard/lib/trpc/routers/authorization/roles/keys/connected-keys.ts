import { db, sql } from "@/lib/db";
import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { z } from "zod";

const assignedKeysResponse = z.object({
  items: z.array(z.string()),
  totalCount: z.number().optional(),
});

export const queryRoleKeys = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .input(
    z.object({
      roleId: z.string(),
      limit: z.number().default(3),
    }),
  )
  .output(assignedKeysResponse)
  .query(async ({ ctx, input }) => {
    const workspaceId = ctx.workspace.id;
    const { roleId, limit } = input;

    const result = await db.execute(sql`
      SELECT 
        CASE 
          WHEN k.name IS NULL OR k.name = '' THEN 
            CONCAT(SUBSTRING(k.id, 1, 8), '...', RIGHT(k.id, 4))
          ELSE k.name 
        END as display_name,
        COUNT(*) OVER() as total_count
      FROM keys_roles kr
      JOIN \`keys\` k ON kr.key_id = k.id
      WHERE kr.role_id = ${roleId} 
        AND kr.workspace_id = ${workspaceId}
      ORDER BY COALESCE(k.name, k.id)
      LIMIT ${limit}
    `);

    const rows = result.rows as {
      display_name: string;
      total_count: number;
    }[];

    if (rows.length === 0) {
      return { items: [] };
    }

    const totalCount = Number(rows[0].total_count);
    const items = rows.map((row) => row.display_name);

    if (totalCount > limit) {
      return {
        items,
        totalCount,
      };
    }

    return { items };
  });
