import { and, count, db, eq } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { keysRoles } from "@unkey/db/src/schema";
import { z } from "zod";

const assignedKeysResponse = z.object({
  totalCount: z.number(),
});

export const queryRoleKeys = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(
    z.object({
      roleId: z.string(),
    }),
  )
  .output(assignedKeysResponse)
  .query(async ({ ctx, input }) => {
    const workspaceId = ctx.workspace.id;
    const { roleId } = input;

    const result = await db
      .select({ count: count() })
      .from(keysRoles)
      .where(and(eq(keysRoles.workspaceId, workspaceId), eq(keysRoles.roleId, roleId)));

    return { totalCount: result[0].count };
  });
