import { db, sql } from "@/lib/db";
import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { z } from "zod";

const permissionsResponse = z.object({
  items: z.array(z.string()),
  totalCount: z.number().optional(),
});

export const queryRolePermissions = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .input(
    z.object({
      roleId: z.string(),
      limit: z.number().default(3),
    }),
  )
  .output(permissionsResponse)
  .query(async ({ ctx, input }) => {
    const workspaceId = ctx.workspace.id;
    const { roleId, limit } = input;

    const result = await db.execute(sql`
      SELECT 
        p.name,
        COUNT(*) OVER() as total_count
      FROM roles_permissions rp
      JOIN permissions p ON rp.permission_id = p.id
      WHERE rp.role_id = ${roleId} 
        AND rp.workspace_id = ${workspaceId}
        AND p.name IS NOT NULL
      ORDER BY p.name
      LIMIT ${limit}
    `);

    const rows = result.rows as {
      name: string;
      total_count: number;
    }[];

    if (rows.length === 0) {
      return { items: [] };
    }

    const totalCount = Number(rows[0].total_count);
    const items = rows.map((row) => row.name);

    if (totalCount > limit) {
      return {
        items,
        totalCount,
      };
    }

    return { items };
  });
